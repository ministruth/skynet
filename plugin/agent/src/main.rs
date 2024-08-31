use base64::{engine::general_purpose::STANDARD, Engine};
use byte_unit::{Byte, UnitType};
use clap::{command, Args, Parser, Subcommand};
use client::run;
use logger::start_logger;
use sha3::{digest::ExtendableOutput, Shake256};
use skynet_api::actix_cloud::tokio;
use skynet_api_monitor::ecies::PublicKey;
use std::{env::consts, fs, net::IpAddr, path::PathBuf};
use sysinfo::{
    CpuRefreshKind, Disks, MemoryRefreshKind, NetworkData, Networks, RefreshKind, System,
};

mod client;
mod logger;
mod shell;
mod socket;

#[derive(Parser, Clone)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,

    /// Show verbose/debug log.
    #[arg(short, long, global = true)]
    verbose: bool,

    /// Do not print any log.
    #[arg(short, long, global = true)]
    quiet: bool,

    /// Use JSON to format log.
    #[arg(long, global = true)]
    log_json: bool,
}

#[derive(Args, Clone)]
pub struct RunArgs {
    /// Server or listen address.
    addr: String,

    /// Connect certificate.
    cert: PathBuf,

    /// Passive mode, wait for server to connect.
    #[arg(short, long)]
    passive: bool,

    /// Override public ip.
    #[arg(long)]
    ip: Option<IpAddr>,

    /// Status report rate (seconds).
    #[arg(long, default_value = "1")]
    report_rate: u32,

    /// Whether to disable shell feature.
    #[arg(long)]
    disable_shell: bool,

    /// Max wait time when retrying.
    #[arg(long, default_value = "16")]
    max_time: u32,

    /// Disk name, match first, sum all specified.
    #[arg(short, long, required = true)]
    disk: Vec<String>,

    /// Interface name, sum all specified.
    #[arg(short, long, required = true)]
    interface: Vec<String>,

    /// Whether to restart on update.
    #[arg(long, default_value = "false")]
    restart: bool,
}

#[derive(Subcommand, Clone)]
enum Commands {
    /// Run agent.
    Run(RunArgs),
    /// List interface in agent.
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
    println!("CPU: {:.2}%", sys.global_cpu_usage());
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
    let (_, _guard) = start_logger(!cli.quiet, cli.log_json, cli.verbose);
    match cli.command {
        Commands::Run(args) => {
            let cert = STANDARD
                .decode(fs::read_to_string(&args.cert).unwrap())
                .unwrap();
            let pubkey = PublicKey::parse(&cert.try_into().unwrap()).unwrap();
            let disks: Vec<String> = Disks::new_with_refreshed_list()
                .into_iter()
                .map(|x| x.name().to_string_lossy().to_string())
                .collect();
            for i in &args.disk {
                assert!(disks.contains(i), "Disk name `{i}` not found");
            }
            let interfaces: Vec<String> = Networks::new_with_refreshed_list()
                .into_iter()
                .filter(|x| !x.1.mac_address().is_unspecified())
                .map(|x| x.0.to_owned())
                .collect();
            for i in &args.interface {
                assert!(interfaces.contains(i), "Interface name `{i}` not found");
            }
            run(args, pubkey).await;
        }
        Commands::List => list(),
    }
}
