use std::{io, path::PathBuf};

use actix_cloud::{self, tracing::error, utils};
use clap::{Args, Parser, Subcommand};
use cmd::{check, run, user};
use logger::start_logger;

include!(concat!(env!("OUT_DIR"), "/response.rs"));

mod api;
mod cmd;
mod db;
mod handler;
mod logger;
mod plugin;
mod request;

#[derive(Parser, Clone)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,

    /// Config file.
    #[arg(
        short,
        long,
        global = true,
        value_name = "FILE",
        default_value = "conf.yml"
    )]
    config: PathBuf,

    /// Plugin folder path.
    #[arg(
        short,
        long,
        global = true,
        value_name = "PATH",
        default_value = "plugin"
    )]
    plugin: PathBuf,

    /// Show verbose/debug log.
    #[arg(short, long, global = true)]
    verbose: bool,

    /// Do not print any log.
    #[arg(short, long, global = true)]
    quiet: bool,

    /// Persist previous session when initialized.
    #[arg(long, global = true)]
    persist_session: bool,

    /// Use JSON to format log.
    #[arg(long, global = true)]
    log_json: bool,
}

#[derive(Subcommand, Clone)]
enum Commands {
    /// Run skynet.
    Run {
        /// Do not print the cover (ascii picture) when start.
        #[arg(long)]
        skip_cover: bool,

        /// Disable CSRF protection, for debugging purpose only.
        #[arg(long)]
        disable_csrf: bool,

        /// Use daemon mode, auto restart on shutdown.
        #[arg(short, long)]
        daemon: bool,
    },
    /// User management.
    User(UserCli),
    /// Check config file.
    Check,
}

#[derive(Args, Clone)]
struct UserCli {
    #[command(subcommand)]
    command: UserCommands,
}

#[derive(Subcommand, Clone)]
enum UserCommands {
    /// Add new user.
    Add {
        /// User avatar.
        #[arg(short, long, value_name = "FILE")]
        avatar: Option<PathBuf>,

        /// New username.
        username: String,
    },

    /// Init root user.
    Init {
        /// User avatar.
        #[arg(short, long, value_name = "FILE")]
        avatar: Option<PathBuf>,
    },

    /// Reset user.
    Reset {
        /// Reset username.
        username: String,
    },
}

#[actix_cloud::main]
async fn main() -> io::Result<()> {
    let cli = Cli::parse();
    let (logger, _guard) = start_logger(!cli.quiet, cli.log_json, cli.verbose);

    let mut restart = false;
    match &cli.command {
        Commands::Run {
            skip_cover,
            disable_csrf,
            daemon,
        } => {
            restart = *daemon;
            run::command(&cli, logger, *skip_cover, *disable_csrf).await;
        }
        Commands::User(user_cli) => user::command(&cli, logger, user_cli).await,
        Commands::Check => check::command(&cli),
    }
    if restart {
        if let Err(e) = utils::restart() {
            error!(error = %e, "Failed to restart program");
        }
    }
    Ok(())
}
