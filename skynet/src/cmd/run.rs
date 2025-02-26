use actix_cloud::{
    actix_web::{
        App, HttpMessage, HttpServer,
        cookie::{Key, SameSite},
        dev::{ServiceRequest, ServiceResponse, fn_service},
        middleware::{self, from_fn},
        web::{Data, scope},
    },
    build_router, csrf,
    logger::Logger,
    request,
    security::SecurityHeader,
    session::{SessionMiddleware, config::PersistentSession},
    state::GlobalState,
    tracing::{info, warn},
    tracing_actix_web::TracingLogger,
    utils,
};
use actix_files::NamedFile;
use qstring::QString;
use skynet_api::{
    Result, Skynet,
    config::CONFIG_SESSION_KEY,
    sea_orm::{DatabaseConnection, TransactionTrait},
    tracing::debug,
    viewer::settings::SettingViewer,
};

use super::init;
use crate::{
    Cli, api,
    request::{
        CSRF_COOKIE, CSRF_HEADER, RealIP, TRACE_HEADER, TracingMiddleware, check_csrf_token,
        error_middleware, wrap_router,
    },
    service,
};

fn print_cover() {
    println!("            __                         __   ");
    println!("      _____|  | _____.__. ____   _____/  |_ ");
    println!("     /  ___/  |/ <   |  |/    \\_/ __ \\   __\\");
    println!("     \\___ \\|    < \\___  |   |  \\  ___/|  |  ");
    println!("    /____  >__|_ \\/ ____|___|  /\\___  >__|  ");
    println!("         \\/     \\/\\/         \\/     \\/      \n");
}

async fn get_session_key(db: &DatabaseConnection) -> Result<String> {
    if let Some(x) = SettingViewer::get(db, CONFIG_SESSION_KEY).await? {
        debug!("Session key is existed, using the previous one");
        Ok(x)
    } else {
        info!("Session key not found, generating new one");
        let key = utils::rand_string(64);
        let tx = db.begin().await?;
        SettingViewer::set(&tx, CONFIG_SESSION_KEY, &key).await?;
        tx.commit().await?;
        Ok(key)
    }
}

fn get_session_middleware(key: &str, skynet: &Skynet, state: &GlobalState) -> SessionMiddleware {
    let cookie_prefix = skynet.config.session.prefix.to_owned();
    let cookie_fn = move |x: &str| format!("{cookie_prefix}{x}");
    SessionMiddleware::builder(state.memorydb.clone(), Key::from(key.as_bytes()))
        .cache_keygen(cookie_fn)
        .cookie_name(skynet.config.session.cookie.to_owned())
        .cookie_secure(skynet.config.listen.ssl)
        .cookie_same_site(SameSite::Strict)
        .session_lifecycle(PersistentSession::default())
        .build()
}

pub async fn command(cli: &Cli, logger: Option<Logger>, skip_cover: bool, disable_csrf: bool) {
    if !skip_cover {
        print_cover();
    }

    let (skynet, state, db, plugin_manager, webpush_manager) = init(cli, logger).await;

    if disable_csrf {
        warn!("CSRF protection is disabled, for debugging purpose only");
    }
    if !skynet.config.recaptcha.enable {
        warn!("Recaptcha is disabled, for debugging purpose only")
    }
    if !skynet.config.listen.ssl && !skynet.config.proxy.enable {
        warn!("SSL is not enabled, your traffic is at risk")
    }
    if !cli.persist_session {
        debug!("Flushing memory sessions");
        state.memorydb.flush().await.unwrap();
    } else {
        debug!("Keeping memory sessions");
    }
    let geoip = if skynet.config.geoip.enable {
        debug!("Parsing geoip file");
        Some(maxminddb::Reader::open_readfile(&skynet.config.geoip.database).unwrap())
    } else {
        debug!("geoip is disabled");
        None
    };
    let session_key = get_session_key(&db).await.unwrap();

    let mut worker = skynet.config.listen.worker;
    if worker == 0 {
        worker = num_cpus::get_physical();
    }
    let mut front_header = SecurityHeader::default();
    if skynet.config.listen.ssl {
        front_header.set_default_hsts();
    }
    let back_header = front_header.clone();
    front_header.content_security_policy = skynet.config.header.csp.clone();

    // run server
    let state = state.build();
    let skynet = Data::new(skynet);
    let cli_data = Data::new(cli.clone());
    let db = Data::new(db);
    let plugin_manager = Data::new(plugin_manager);
    let geoip = Data::new(geoip);
    let webpush_manager = Data::new(webpush_manager);
    let mut route = api::new_api(&skynet.default_id);
    service::init_handler(skynet.clone(), state.clone());
    route = plugin_manager.register(&skynet, route).await;

    let server = HttpServer::new({
        let state = state.clone();
        let skynet = skynet.clone();
        move || {
            App::new()
                .service(
                    scope("/api")
                        .configure(build_router(
                            wrap_router(route.clone(), disable_csrf),
                            csrf::Middleware::new(
                                CSRF_COOKIE.into(),
                                CSRF_HEADER.into(),
                                check_csrf_token(skynet.config.csrf.prefix.clone()),
                            ),
                        ))
                        .wrap(get_session_middleware(&session_key, &skynet, &state))
                        .wrap(back_header.clone().build()),
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
                .wrap(front_header.clone().build())
                .wrap(
                    request::Middleware::new()
                        .trace_header(TRACE_HEADER)
                        .real_ip(|req| req.extensions().get::<RealIP>().unwrap().0.to_owned())
                        .lang(|req| {
                            QString::from(req.query_string())
                                .get("lang")
                                .map(ToOwned::to_owned)
                        }),
                )
                .wrap(from_fn(error_middleware))
                .wrap(TracingLogger::<TracingMiddleware>::new())
                .wrap(middleware::Compress::default())
                .app_data(state.clone())
                .app_data(skynet.clone())
                .app_data(cli_data.clone())
                .app_data(db.clone())
                .app_data(plugin_manager.clone())
                .app_data(geoip.clone())
                .app_data(webpush_manager.clone())
        }
    })
    .workers(worker);

    let address = &skynet.config.listen.address;
    let server = if skynet.config.listen.ssl {
        server
            .bind_rustls_0_23(
                address,
                utils::load_rustls_config(
                    skynet.config.listen.ssl_cert.clone().unwrap(),
                    skynet.config.listen.ssl_key.clone().unwrap(),
                )
                .unwrap(),
            )
            .unwrap()
            .run()
    } else {
        server.bind(address).unwrap().run()
    };
    info!("Listening on {address}");
    state.server.start(server).await.unwrap();
}
