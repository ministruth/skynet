use actix_cloud::tracing::info;
use skynet_api::config;

use crate::Cli;

pub fn command(cli: &Cli) {
    let _ = config::load_file(cli.config.to_str().unwrap());
    info!("Config file {:?} valid", cli.config);
}
