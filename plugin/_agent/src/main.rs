use std::{io, time};

use byte_unit::{Byte, UnitType};
use clap::{command, Parser, Subcommand};
use client::run;
use fern::colors::{Color, ColoredLevelConfig};
use log::LevelFilter;
use serde_json::json;
use sha3::{digest::ExtendableOutput, Shake256};
use sysinfo::{
    CpuRefreshKind, Disks, MemoryRefreshKind, NetworkData, Networks, RefreshKind, System,
};

mod client;

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
    },
    /// List interface in agent
    List,
}

fn init_logger(verbose: bool, json: bool) {
    let level_color = ColoredLevelConfig::new()
        .debug(Color::BrightMagenta)
        .info(Color::BrightBlue)
        .warn(Color::BrightYellow)
        .error(Color::BrightRed);
    let mut logger = fern::Dispatch::new().level(if verbose {
        LevelFilter::Debug
    } else {
        LevelFilter::Info
    });
    logger = logger.format(move |out, message, record| {
        if json {
            let time = time::SystemTime::now()
                .duration_since(time::UNIX_EPOCH)
                .unwrap()
                .as_secs();
            out.finish(format_args!(
                "{}",
                serde_json::to_string(&json!({
                    "time":time,
                    "level":record.level().as_str(),
                    "msg":message,
                }))
                .unwrap()
            ));
        } else {
            out.finish(format_args!(
                "{}[{}] {}",
                chrono::Local::now().format("[%Y-%m-%d %H:%M:%S]"),
                level_color.color(record.level()),
                message
            ));
        }
    });
    logger = logger
        .chain(
            fern::Dispatch::new()
                .filter(|s| s.level() != LevelFilter::Error)
                .chain(io::stdout()),
        )
        .chain(
            fern::Dispatch::new()
                .level(LevelFilter::Error)
                .chain(io::stderr()),
        );
    logger.apply().unwrap();
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
    println!(
        "Machine:  {}",
        System::cpu_arch().unwrap_or_else(|| "N/A".to_owned())
    );
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

#[tokio::main]
async fn main() {
    let cli = Cli::parse();
    if !cli.quiet {
        init_logger(cli.verbose, cli.log_json);
    }
    match cli.command {
        Commands::Run {
            addr,
            token,
            max_time,
        } => run(addr, token, max_time).await,
        Commands::List => list(),
    }
}
