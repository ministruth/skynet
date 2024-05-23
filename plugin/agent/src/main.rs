use byte_unit::{Byte, UnitType};
use chrono::{DateTime, Local, Utc};
use clap::{command, Parser, Subcommand};
use client::run;
use sha3::{digest::ExtendableOutput, Shake256};
use skynet::{
    logger::{LogItem, LogSender, Logger},
    tracing::Level,
    tracing_subscriber,
};
use std::{
    env::consts,
    io::{self, stderr, stdout},
    str::FromStr,
    sync::mpsc,
    thread,
};
use sysinfo::{
    CpuRefreshKind, Disks, MemoryRefreshKind, NetworkData, Networks, RefreshKind, System,
};

mod client;
mod shell;
mod socket;

#[derive(Parser, Clone)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,

    /// Show verbose/debug log
    #[arg(short, long, global = true)]
    verbose: bool,

    /// Do not print any log
    #[arg(short, long, global = true)]
    quiet: bool,

    /// Use JSON to format log
    #[arg(long, global = true)]
    log_json: bool,
}

#[derive(Subcommand, Clone)]
enum Commands {
    /// Run agent
    Run {
        /// Server address
        addr: String,

        /// Connect token
        token: String,

        /// Max wait time when retrying
        #[arg(long, default_value = "16")]
        max_time: u32,

        /// Disk name, match first, sum all specified.
        #[arg(short, long, required = true)]
        disk: Vec<String>,

        /// Interface name, sum all specified.
        #[arg(short, long, required = true)]
        interface: Vec<String>,
    },
    /// List interface in agent
    List,
}

fn get_uid() -> String {
    let mut id = [0_u8; 16];
    Shake256::digest_xof(machine_uid::get().unwrap(), &mut id);
    hex::encode(id)
}

fn list() {
    let mut sys = System::new();
    sys.refresh_specifics(
        RefreshKind::new()
            .with_cpu(CpuRefreshKind::everything())
            .with_memory(MemoryRefreshKind::everything()),
    );

    println!("UID:      {}", get_uid());
    println!(
        "OS:       {}",
        System::name().unwrap_or_else(|| "N/A".to_owned())
    );
    println!("Arch:     {}", consts::ARCH);
    println!(
        "System:   {}",
        System::long_os_version().unwrap_or_else(|| "N/A".to_owned())
    );
    println!(
        "Hostname: {}",
        System::host_name().unwrap_or_else(|| "N/A".to_owned())
    );

    println!();
    println!("CPU: {:.2}%", sys.global_cpu_info().cpu_usage());
    println!(
        "Memory: {:.2} / {:.2}",
        Byte::from_u64(sys.used_memory()).get_appropriate_unit(UnitType::Binary),
        Byte::from_u64(sys.total_memory()).get_appropriate_unit(UnitType::Binary)
    );

    println!("\nDisks:");
    let disks = Disks::new_with_refreshed_list();
    for disk in &disks {
        println!(
            "[{}][{}][{:?}] -> {} ({:.2} / {:.2})",
            disk.name().to_string_lossy(),
            disk.file_system().to_string_lossy(),
            disk.kind(),
            disk.mount_point().to_string_lossy(),
            Byte::from_u64(disk.total_space() - disk.available_space())
                .get_appropriate_unit(UnitType::Binary),
            Byte::from_u64(disk.total_space()).get_appropriate_unit(UnitType::Binary)
        );
    }

    println!("\nInterfaces:");
    let networks = Networks::new_with_refreshed_list();
    let mut networks: Vec<(&String, &NetworkData)> = networks
        .into_iter()
        .filter(|x| !x.1.mac_address().is_unspecified())
        .collect();
    networks.sort_by_key(|x| x.0);
    for (name, data) in &networks {
        println!(
            "[{name}][{}] TX/RX: {:.2} / {:.2}",
            data.mac_address(),
            Byte::from_u64(data.total_transmitted()).get_appropriate_unit(UnitType::Binary),
            Byte::from_u64(data.total_received()).get_appropriate_unit(UnitType::Binary)
        );
    }
}

fn start_logger(json: bool, verbose: bool) {
    let (tx, rx) = mpsc::channel();
    tracing_subscriber::fmt()
        .with_max_level(if verbose { Level::DEBUG } else { Level::INFO })
        .with_writer(LogSender::new(tx))
        .without_time()
        .with_file(true)
        .with_line_number(true)
        .json()
        .init();

    thread::spawn(move || {
        while let Ok(v) = rx.recv() {
            let mut item = LogItem::from_json(v);
            let level = Level::from_str(&item.level).unwrap_or(Level::ERROR);
            if item.target.starts_with("tungstenite::") {
                continue;
            }
            item.filename = Logger::trim_filename(&item.filename);
            let time = item.fields.remove("_time").unwrap_or_default().as_i64();
            let token: String = item
                .fields
                .remove("_token")
                .unwrap_or_default()
                .as_str()
                .unwrap_or("main")
                .chars()
                .take(8)
                .collect();
            if !verbose {
                item.filename.clear();
                item.line_number = 0;
            }

            let writer: Box<dyn io::Write> = if level <= Level::WARN {
                Box::new(stderr())
            } else {
                Box::new(stdout())
            };
            if json {
                item.target.clear();
                if token != "main" {
                    item.fields.insert(String::from("token"), token.into());
                }
                item.time = time.unwrap_or_else(|| Utc::now().timestamp_micros()).into();
                let _ = item.write_json(writer);
            } else {
                item.level = Logger::fmt_level(&level);
                item.target = token; // we do not use target, show token instead.
                item.time = time
                    .map_or_else(Local::now, |v| {
                        DateTime::from_timestamp_micros(v)
                            .unwrap_or_default()
                            .into()
                    })
                    .format("%F %T%.6f")
                    .to_string()
                    .into();
                let _ = item.write_console(writer);
            }
        }
    });
}

#[tokio::main]
async fn main() {
    let cli = Cli::parse();
    if !cli.quiet {
        start_logger(cli.log_json, cli.verbose);
    }
    match cli.command {
        Commands::Run {
            addr,
            token,
            max_time,
            disk,
            interface,
        } => {
            let disks: Vec<String> = Disks::new_with_refreshed_list()
                .into_iter()
                .map(|x| x.name().to_string_lossy().to_string())
                .collect();
            for i in &disk {
                assert!(disks.contains(i), "Disk name `{i}` not found");
            }
            let interfaces: Vec<String> = Networks::new_with_refreshed_list()
                .into_iter()
                .filter(|x| !x.1.mac_address().is_unspecified())
                .map(|x| x.0.to_owned())
                .collect();
            for i in &interface {
                assert!(interfaces.contains(i), "Interface name `{i}` not found");
            }
            run(addr, token, max_time, disk, interface).await;
        }
        Commands::List => list(),
    }
}
