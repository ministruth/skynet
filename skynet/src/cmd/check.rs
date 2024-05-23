use skynet::config;
use tracing::info;

use crate::Cli;

pub fn command(cli: &Cli) {
    let _ = config::load_file(cli.config.to_str().unwrap());
    info!("Config file {:?} valid", cli.config);
}
