use actix_files::NamedFile;
use qstring::QString;
use skynet_api::{
    actix_cloud::{
        actix_web::{
            cookie::{time::Duration, Key, SameSite},
            dev::{fn_service, ServiceRequest, ServiceResponse},
            middleware::{self, from_fn},
            web::{scope, Data},
            App, HttpMessage, HttpServer,
        },
        build_router, csrf, request,
        security::SecurityHeader,
        session::{
            config::{PersistentSession, TtlExtensionPolicy},
            SessionMiddleware,
        },
        state::GlobalState,
        tracing_actix_web::TracingLogger,
        utils,
    },
    logger::Logger,
    tracing::{info, warn},
    Skynet,
};

use super::init;
use crate::{
    api,
    request::{
        check_csrf_token, error_middleware, RealIP, TracingMiddleware, CSRF_COOKIE, CSRF_HEADER,
        TRACE_HEADER,
    },
    Cli,
};

fn print_cover() {
    println!("            __                         __   ");
    println!("      _____|  | _____.__. ____   _____/  |_ ");
    println!("     /  ___/  |/ <   |  |/    \\_/ __ \\   __\\");
    println!("     \\___ \\|    < \\___  |   |  \\  ___/|  |  ");
    println!("    /____  >__|_ \\/ ____|___|  /\\___  >__|  ");
    println!("         \\/     \\/\\/         \\/     \\/      \n");
}

fn get_session_middleware(skynet: &Skynet, state: &GlobalState) -> SessionMiddleware {
    let cookie_prefix = skynet.config.session.prefix.to_owned();
    let cookie_fn = move |x: &str| format!("{cookie_prefix}{x}");
    SessionMiddleware::builder(
        state.memorydb.clone(),
        Key::from(skynet.config.session.key.as_bytes()),
    )
    .cache_keygen(cookie_fn)
    .cookie_name(skynet.config.session.cookie.to_owned())
    .cookie_secure(skynet.config.listen.ssl)
    .cookie_same_site(SameSite::Strict)
    .session_lifecycle(
        PersistentSession::default()
            .session_ttl_extension_policy(TtlExtensionPolicy::OnEveryRequest)
            .session_ttl(Duration::seconds(skynet.config.session.expire.into())),
    )
    .build()
}

pub async fn command(cli: &Cli, logger: Logger, skip_cover: bool, disable_csrf: bool) {
    if !skip_cover {
        print_cover();
    }

    let (skynet, state, db) = init(cli, logger).await;

    if disable_csrf {
        warn!("CSRF protection is disabled, for debugging purpose only");
    }
    if !skynet.config.listen.ssl && !skynet.config.proxy.enable {
        warn!("SSL is not enabled, your traffic is at risk")
    }
    if !cli.persist_session {
        state.memorydb.flush().await.unwrap();
    }

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
    let server = HttpServer::new({
        let state = state.clone();
        let skynet = skynet.clone();
        move || {
            let mut route = api::new_api(&skynet.default_id, disable_csrf);
            route = skynet.plugin.parse_route(&skynet, route);

            App::new()
                .service(
                    scope("/api")
                        .configure(build_router(
                            route,
                            csrf::Middleware::new(
                                CSRF_COOKIE.into(),
                                CSRF_HEADER.into(),
                                check_csrf_token(skynet.config.csrf.prefix.clone()),
                            ),
                        ))
                        .wrap(get_session_middleware(&skynet, &state))
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
