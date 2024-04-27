use std::{
    collections::HashMap,
    env, io,
    path::PathBuf,
    process::Command,
    sync::{atomic::AtomicU64, Arc},
};

use chrono::DateTime;
use clap::{Args, Parser, Subcommand};
use cmd::{check, run, user};
use enum_as_inner::EnumAsInner;
use enum_map::EnumMap;
use handler_impl::{
    group::DefaultGroupHandler, notifications::DefaultNotificationHandler,
    permission::DefaultPermHandler, setting::DefaultSettingHandler, user::DefaultUserHandler,
};
use parking_lot::RwLock;
use skynet::{config::Config, logger::Logger, plugin::PluginManager};

mod api;
mod cmd;
mod handler_impl;

#[allow(clippy::struct_excessive_bools)]
#[derive(Parser, Clone)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,

    /// Config file
    #[arg(
        short,
        long,
        global = true,
        value_name = "FILE",
        default_value = "conf.yml"
    )]
    config: PathBuf,

    /// Plugin folder path
    #[arg(
        short,
        long,
        global = true,
        value_name = "PATH",
        default_value = "plugin"
    )]
    plugin: PathBuf,

    /// Show verbose/debug log
    #[arg(short, long, global = true)]
    verbose: bool,

    /// Do not print any log
    #[arg(short, long, global = true)]
    quiet: bool,

    /// Persist previous session when initialized
    #[arg(long, global = true)]
    persist_session: bool,

    /// Use JSON to format log
    #[arg(long, global = true)]
    log_json: bool,
}

#[derive(Subcommand, EnumAsInner, Clone)]
enum Commands {
    /// Run skynet
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
    /// User management
    User(UserCli),
    /// Check config file
    Check,
}

#[derive(Args, Clone)]
struct UserCli {
    #[command(subcommand)]
    command: UserCommands,
}

#[derive(Subcommand, EnumAsInner, Clone)]
enum UserCommands {
    /// Add new user
    Add {
        /// User avatar
        #[arg(short, long, value_name = "FILE")]
        avatar: Option<PathBuf>,

        /// New username
        username: String,
    },

    /// Init root user
    Init {
        /// User avatar
        #[arg(short, long, value_name = "FILE")]
        avatar: Option<PathBuf>,
    },

    /// Reset user
    Reset {
        /// Reset username
        username: String,
    },
}

#[actix_web::main]
async fn main() -> io::Result<()> {
    let cli = Cli::parse();
    let mut skynet = skynet::Skynet {
        logger: Logger::new(),
        user: Box::new(DefaultUserHandler::new()),
        group: Box::new(DefaultGroupHandler::new()),
        perm: Box::new(DefaultPermHandler::new()),
        notification: Box::new(DefaultNotificationHandler::new()),
        setting: Box::new(DefaultSettingHandler::new()),

        default_id: EnumMap::default(),
        config: Config::new(),
        locale: HashMap::new(),

        plugin: PluginManager::new(),
        menu: Vec::new(),

        unread_notification: Arc::new(AtomicU64::new(0)),
        running: RwLock::new(false),
        start_time: DateTime::default(),

        shared_api: HashMap::new(),
    };
    // init logger first
    skynet
        .logger
        .init(
            skynet.unread_notification.clone(),
            !cli.quiet,
            cli.log_json,
            cli.verbose,
        )
        .unwrap();

    let mut restart = false;
    match &cli.command {
        Commands::Run {
            skip_cover,
            disable_csrf,
            daemon,
        } => {
            restart = *daemon;
            Box::pin(run::command(&cli, skynet, *skip_cover, *disable_csrf)).await;
        }
        Commands::User(user_cli) => Box::pin(user::command(&cli, skynet, user_cli)).await,
        Commands::Check => check::command(&cli),
    }
    if restart {
        return Command::new(env::current_exe().unwrap())
            .args(env::args().skip(1))
            .spawn()
            .map_err(Into::into)
            .and(Ok(()));
    }
    Ok(())
}
