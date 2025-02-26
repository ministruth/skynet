use std::sync::Arc;

use actix_cloud::{
    self,
    i18n::{Locale, i18n},
    memorydb::{MemoryDB, default::DefaultBackend, redis::RedisBackend},
    state::{GlobalState, ServerHandle},
    tracing::debug,
};
use skynet_api::{
    Skynet, config, ffi_rpc::registry::Registry, logger::Logger, sea_orm::DatabaseConnection,
    service::SKYNET_SERVICE,
};

use crate::{
    Cli, api, db, logger,
    plugin::PluginManager,
    service::{self, ServiceImpl},
    webpush::WebpushManager,
};

pub mod check;
pub mod run;
pub mod user;

async fn init(
    cli: &Cli,
    logger: Option<actix_cloud::logger::Logger>,
) -> (
    Skynet,
    GlobalState,
    DatabaseConnection,
    PluginManager,
    WebpushManager,
) {
    // load config
    let (state_config, config) = config::load_file(cli.config.to_str().unwrap());
    debug!("Config file {:?} loaded", cli.config);

    // init locale
    let locale = Locale::new(config.lang.clone()).add_locale(i18n!("locales"));
    debug!("Locale loaded: {}", locale.locale.len());

    // init memorydb
    let memorydb: Arc<dyn MemoryDB> = if config.redis.enable {
        Arc::new(
            RedisBackend::new(&config.redis.dsn.clone().unwrap())
                .await
                .expect("Redis connect failed"),
        )
    } else {
        Arc::new(DefaultBackend::new())
    };
    debug!("Memorydb connected");

    // init db
    let (db, default_id) = db::init(&config.database.dsn)
        .await
        .expect("DB connect failed");

    // init webpush
    let webpush = WebpushManager::new(db.clone(), &default_id).await.unwrap();

    // init notification
    logger::set_db(db.clone()).await;
    logger::set_webpush(webpush.clone()).await;

    let skynet = Skynet {
        default_id,
        config,
        logger: Logger {
            verbose: cli.verbose,
            json: cli.log_json,
            enable: !cli.quiet,
        },
        menu: api::new_menu(&default_id),
        warning: Default::default(),
    };
    let state = GlobalState {
        memorydb,
        config: state_config,
        logger,
        locale,
        server: ServerHandle::default(),
    };
    let mut reg = Registry::default();
    ServiceImpl::register_mock(&mut reg, SKYNET_SERVICE);

    // init service
    service::init(&state, webpush.clone());

    // init plugin
    let mut plugin = PluginManager::new(reg);
    let mut skynet = plugin.load_all(&db, skynet, &cli.plugin).await;

    // set menu badge, must after plugin init.
    api::set_menu_badge(&mut skynet);

    (skynet, state, db, plugin, webpush)
}
