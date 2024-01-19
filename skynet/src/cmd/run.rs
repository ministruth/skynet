use std::{fs::File, io::BufReader, path::Path};

use crate::{api, db, Cli};
use actix_files::NamedFile;
use actix_session::{
    config::{PersistentSession, TtlExtensionPolicy},
    storage::RedisSessionStore,
    SessionMiddleware,
};
use actix_web::{
    cookie::{time::Duration, Key, SameSite},
    dev::{fn_service, ServerHandle, ServiceRequest, ServiceResponse},
    middleware, web, App, HttpServer,
};
use actix_web_validator::JsonConfig;
use chrono::Utc;
use log::{debug, warn};
use parking_lot::Mutex;
use redis::aio::ConnectionManager;
use rustls::{Certificate, PrivateKey, ServerConfig};
use rustls_pemfile::{certs, pkcs8_private_keys};
use sea_orm::TransactionTrait;
use skynet::{config, logger, permission::DEFAULT_ID, plugin::PluginManager, Skynet};
use skynet_i18n::i18n;

pub async fn init_skynet(cli: &Cli, mut skynet: Skynet) -> Skynet {
    // load config
    skynet.config = config::load_file(cli.config.to_str().unwrap());
    debug!("Config file {:?} loaded", cli.config);

    // locale
    skynet.add_locale(i18n!("locales"));

    // init db
    skynet.db = db::connect(skynet.config.database_dsn.get()).await.unwrap();
    DEFAULT_ID.set(db::init(&skynet.db).await.unwrap()).unwrap();
    debug!("DB connected");

    // init redis
    skynet.redis = Some(
        ConnectionManager::new(redis::Client::open(skynet.config.redis_dsn.get()).unwrap())
            .await
            .unwrap(),
    );
    debug!("Redis connected");

    // init notification
    logger::DBLOGGER.set(skynet.db.clone()).unwrap();
    logger::Logger::write_db_pending(&skynet.db).await;

    // init setting
    let tx = skynet.db.begin().await.unwrap();
    skynet.setting.build_cache(&tx).await.unwrap();
    tx.commit().await.unwrap();

    // init menu
    skynet.menu = api::new_menu();

    // init plugin
    let mut plugin = PluginManager::new();
    let mut skynet = plugin.load_all(skynet, &cli.plugin);
    skynet.plugin = plugin;
    skynet
}

fn load_rustls_config<P: AsRef<Path>>(cert: P, key: P) -> ServerConfig {
    let config = ServerConfig::builder()
        .with_safe_defaults()
        .with_no_client_auth();
    let cert_chain = certs(&mut BufReader::new(File::open(cert).unwrap()))
        .unwrap()
        .into_iter()
        .map(Certificate)
        .collect();
    let mut keys: Vec<PrivateKey> =
        pkcs8_private_keys(&mut BufReader::new(File::open(key).unwrap()))
            .unwrap()
            .into_iter()
            .map(PrivateKey)
            .collect();

    assert!(!keys.is_empty(), "Could not locate PKCS 8 private keys");
    config.with_single_cert(cert_chain, keys.remove(0)).unwrap()
}

fn get_security_header(ssl: bool, csp: String) -> middleware::DefaultHeaders {
    let mut ret = middleware::DefaultHeaders::new()
        .add(("X-Content-Type-Options", "nosniff"))
        .add(("Referrer-Policy", "same-origin"))
        .add(("X-Frame-Options", "DENY"))
        .add(("X-XSS-Protection", "1; mode=block"))
        .add(("Cross-Origin-Opener-Policy", "same-origin"))
        .add(("Referrer-Policy", "same-origin"))
        .add(("Content-Security-Policy", csp));
    if ssl {
        ret = ret.add(("Strict-Transport-Security", "max-age=31536000; preload"));
    }
    ret
}

fn get_session_middleware(s: &Skynet) -> SessionMiddleware<RedisSessionStore> {
    let cookie_prefix = s.config.session_prefix.get().to_owned();
    let cookie_fn = move |x: &str| format!("{cookie_prefix}{x}");
    SessionMiddleware::builder(
        RedisSessionStore::builder(s.redis.clone().unwrap())
            .cache_keygen(cookie_fn.clone())
            .build()
            .unwrap(),
        Key::from(s.config.session_key.get().as_bytes()),
    )
    .cookie_name(s.config.session_cookie.get().to_owned())
    .cookie_secure(s.config.listen_ssl.get())
    .cookie_same_site(SameSite::Strict)
    .session_lifecycle(
        PersistentSession::default()
            .session_ttl_extension_policy(TtlExtensionPolicy::OnEveryRequest)
            .session_ttl(Duration::seconds(s.config.session_expire.get())),
    )
    .build()
}

#[derive(Default)]
pub struct StopHandle {
    inner: Mutex<Option<ServerHandle>>,
}

impl StopHandle {
    /// Sets the server handle to stop.
    pub fn register(&self, handle: ServerHandle) {
        *self.inner.lock() = Some(handle);
    }

    /// Sends stop signal through contained server handle.
    pub fn stop(&self, graceful: bool) {
        #[allow(clippy::let_underscore_future)]
        let _ = self.inner.lock().as_ref().unwrap().stop(graceful);
    }
}

pub async fn command(cli: &Cli, skynet: Skynet, disable_csrf: bool) {
    let mut skynet = init_skynet(cli, skynet).await;
    if disable_csrf {
        warn!("CSRF protection is disabled, for debugging purpose only");
    }
    if !cli.persist_session {
        let _: () = redis::cmd("FLUSHDB")
            .query_async(skynet.redis.as_mut().unwrap())
            .await
            .unwrap();
    }

    let mut worker: usize = skynet.config.listen_worker.get().try_into().unwrap();
    if worker == 0 {
        worker = num_cpus::get_physical();
    }
    // run server
    skynet.start_time = Utc::now();
    let skynet = web::Data::new(skynet);
    let cli_data = web::Data::new(cli.clone());
    let stop_handle = web::Data::new(StopHandle::default());
    let server = HttpServer::new({
        let stop_handle = stop_handle.clone();
        let skynet = skynet.clone();
        move || {
            let mut route = api::new_api();
            route = skynet.plugin.parse_route(route);

            App::new()
                .service(
                    web::scope("/api")
                        .configure(api::router(route, disable_csrf))
                        .wrap(get_session_middleware(&skynet)),
                )
                .service(
                    actix_files::Files::new("/", "./assets")
                        .index_file("index.html")
                        .default_handler(fn_service(|req: ServiceRequest| async {
                            let (req, _) = req.into_parts();
                            let file = NamedFile::open_async("./assets/index.html").await?;
                            let res = file.into_response(&req);
                            Ok(ServiceResponse::new(req, res))
                        })),
                )
                .wrap(middleware::Compress::default())
                .wrap(get_security_header(
                    skynet.config.listen_ssl.get(),
                    skynet.config.header_csp.get().to_owned(),
                ))
                .wrap(middleware::Logger::default())
                .app_data(skynet.clone())
                .app_data(cli_data.clone())
                .app_data(
                    JsonConfig::default().limit(skynet.config.max_body.get().try_into().unwrap()),
                )
                .app_data(stop_handle.clone())
        }
    })
    .workers(worker);

    let address = skynet.config.listen_address.get();
    let server = if skynet.config.listen_ssl.get() {
        server
            .bind_rustls_021(
                address,
                load_rustls_config(
                    skynet.config.listen_ssl_cert.get(),
                    skynet.config.listen_ssl_key.get(),
                ),
            )
            .unwrap()
            .run()
    } else {
        server.bind(address).unwrap().run()
    };
    stop_handle.register(server.handle());
    *skynet.running.write() = true;
    server.await.unwrap();
}
