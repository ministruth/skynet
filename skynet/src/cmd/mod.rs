use std::sync::Arc;

use actix_cloud::{
    self,
    i18n::{i18n, Locale},
    memorydb::{default::DefaultBackend, redis::RedisBackend, MemoryDB},
    state::{GlobalState, ServerHandle},
    tracing::debug,
};
use enum_map::EnumMap;
use skynet_api::{
    api::APIManager,
    config,
    logger::Logger,
    sea_orm::{DatabaseConnection, TransactionTrait},
    Skynet,
};

use crate::{api, db, handler::*, logger, plugin::PluginManager, Cli};

pub mod check;
pub mod run;
pub mod user;

async fn init(
    cli: &Cli,
    logger: Logger,
) -> (Skynet, GlobalState, DatabaseConnection, PluginManager) {
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

    let mut skynet = Skynet {
        user: Box::new(DefaultUserHandler::new()),
        group: Box::new(DefaultGroupHandler::new()),
        perm: Box::new(DefaultPermHandler::new()),
        notification: Box::new(DefaultNotificationHandler::new()),
        setting: Box::new(DefaultSettingHandler::new()),
        logger,
        default_id: EnumMap::default(),
        config,
        menu: Vec::new(),
        shared_api: APIManager::new(),
    };
    let state = GlobalState {
        memorydb,
        config: state_config,
        logger: skynet.logger.logger.clone(),
        locale,
        server: ServerHandle::default(),
    };

    // init db
    let (db, default_id) = db::init(&skynet).await.expect("DB connect failed");
    skynet.default_id = default_id;
    debug!("DB connected");

    // init notification
    logger::set_db(db.clone()).await;

    // init setting
    let tx = db.begin().await.unwrap();
    skynet.setting.build_cache(&tx).await.unwrap();
    tx.commit().await.unwrap();

    // init menu
    skynet.menu = api::new_menu(&skynet.default_id);

    // init plugin
    let mut plugin = PluginManager::new();
    let (skynet, state) = plugin.load_all(skynet, state, &cli.plugin);

    (skynet, state, db, plugin)
}
