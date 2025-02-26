use std::{
    collections::HashMap, future::Future, net::SocketAddr, pin::Pin, rc::Rc, str::FromStr,
    sync::Arc, time::Duration,
};

use actix_cloud::{
    actix_web::{
        self, HttpMessage, HttpRequest, HttpResponse,
        body::MessageBody,
        dev::{ServiceRequest, ServiceResponse},
        http::{
            StatusCode,
            header::{HeaderName, HeaderValue},
        },
        middleware::Next,
        rt::spawn,
        web::{Bytes, Data, Payload},
    },
    async_trait,
    chrono::Utc,
    request::Extension,
    response::RspResult,
    router::{CSRFType, Checker},
    session::SessionExt,
    state::GlobalState,
    tokio::select,
    tracing::{Span, error, info, info_span},
    tracing_actix_web::{DefaultRootSpanBuilder, RequestId, RootSpanBuilder},
    utils,
};
use actix_ws::Message;
use derivative::Derivative;
use futures::StreamExt;
use skynet_api::{
    HyUuid, Result, Skynet,
    permission::{GUEST_ID, GUEST_NAME, PERM_ALL, PermChecker, PermissionItem},
    plugin::{self, Body, WSMessage},
    request::{Request, Router, RouterType},
    sea_orm::DatabaseConnection,
    tracing::debug,
};

use crate::{
    Cli,
    api::api_call,
    plugin::PluginManager,
    service::{self, websocket::WEBSOCKETIMPL_INSTANCE},
};

pub const CSRF_COOKIE: &str = "CSRF_TOKEN";
pub const CSRF_HEADER: &str = "X-CSRF-Token";
pub const TRACE_HEADER: &str = "x-trace-id"; // This should be lowercase.

#[macro_export]
macro_rules! finish_ok {
    () => {
        skynet_api::finish!(actix_cloud::response::JsonResponse::new(
            $crate::SkynetResponse::Success
        ))
    };
}

#[macro_export]
macro_rules! finish_err {
    ($rsp:path) => {
        skynet_api::finish!(actix_cloud::response::JsonResponse::new($rsp))
    };
}

#[macro_export]
macro_rules! finish_data {
    ($rsp:expr) => {
        skynet_api::finish!(
            actix_cloud::response::JsonResponse::new($crate::SkynetResponse::Success).json($rsp)
        )
    };
}

/// Generate new csrf token.
/// 32 length, a-zA-Z0-9.
pub async fn new_csrf_token(skynet: &Skynet, state: &GlobalState) -> Result<String> {
    let token = utils::rand_string(32);
    state
        .memorydb
        .set_ex(
            &format!("{}{}", skynet.config.csrf.prefix, token),
            "1",
            &Duration::from_secs(skynet.config.csrf.expire.into()),
        )
        .await?;
    Ok(token)
}

/// Check csrf token.
pub fn check_csrf_token(
    prefix: String,
) -> impl Fn(HttpRequest, String) -> Pin<Box<dyn Future<Output = Result<bool, actix_web::Error>>>> {
    let prefix = Rc::new(prefix);
    move |req, token| {
        let value = prefix.clone();
        let fut = async move {
            let key = format!("{}{}", value, token);
            let state = req.app_data::<Data<GlobalState>>().unwrap();
            let res = state
                .memorydb
                .get_del(&key)
                .await
                .map_err(actix_web::error::ErrorBadGateway)?;
            res.map_or_else(|| Ok(false), |x| Ok(x == "1"))
        };
        Box::pin(fut)
    }
}

#[derive(thiserror::Error, Derivative)]
#[derivative(Debug)]
pub enum APIError {
    #[error("Validation error: missing field `{0}`")]
    MissingField(String),
    #[error("Missing setting `{0}`")]
    MissingSetting(String),
    #[error("Invalid setting `{0}`")]
    InvalidSetting(String),
}

/// Eat additional logs to user.
pub async fn error_middleware(
    cli: Data<Cli>,
    req: ServiceRequest,
    next: Next<impl MessageBody + 'static>,
) -> Result<ServiceResponse<impl MessageBody>, actix_web::Error> {
    let start_time = Utc::now();
    match next.call(req).await {
        Ok(rsp) => {
            let process_time = (Utc::now() - start_time).num_milliseconds();
            if let Some(error) = rsp.response().error() {
                error!(
                    code = rsp.status().as_u16(),
                    process_time,
                    error = %error.as_response_error(),
                    "HTTP Request"
                );
            } else {
                info!(code = rsp.status().as_u16(), process_time, "HTTP Request");
            }

            let (req, rsp) = rsp.into_parts();
            if !cli.verbose && (rsp.status().is_client_error() || rsp.status().is_server_error()) {
                Ok(ServiceResponse::new(req, rsp.drop_body()).map_into_left_body())
            } else {
                Ok(ServiceResponse::new(req, rsp).map_into_right_body())
            }
        }
        Err(e) => {
            let rsp = e.error_response();
            let process_time = (Utc::now() - start_time).num_milliseconds();
            error!(
                code = rsp.status().as_u16(),
                process_time,
                error = %e,
                "HTTP Request"
            );

            let body = if !cli.verbose {
                String::new()
            } else {
                e.to_string()
            };
            Err(match rsp.status() {
                StatusCode::BAD_REQUEST => actix_web::error::ErrorBadRequest(body),
                StatusCode::FORBIDDEN => actix_web::error::ErrorForbidden(body),
                StatusCode::NOT_FOUND => actix_web::error::ErrorNotFound(body),
                StatusCode::INTERNAL_SERVER_ERROR => {
                    actix_web::error::ErrorInternalServerError(body)
                }
                _ => e,
            })
        }
    }
}

pub struct RealIP(pub SocketAddr);

fn get_real_ip(req: &ServiceRequest) -> SocketAddr {
    let skynet = req.app_data::<Data<Skynet>>().unwrap();
    let mut ip = req.peer_addr().unwrap();
    if skynet.config.proxy.enable {
        ip = req
            .headers()
            .get(&skynet.config.proxy.header)
            .map(|x| x.to_str().unwrap())
            .unwrap()
            .parse()
            .unwrap();
    }
    req.extensions_mut().insert(RealIP(ip));
    ip
}

pub struct TracingMiddleware;

impl RootSpanBuilder for TracingMiddleware {
    fn on_request_start(request: &ServiceRequest) -> Span {
        let trace_id = request.extensions().get::<RequestId>().unwrap().to_string();
        let user_agent = request
            .headers()
            .get("User-Agent")
            .map_or("", |h| h.to_str().unwrap_or(""))
            .to_owned();
        let method = request.method().to_string();
        let path = request
            .uri()
            .path_and_query()
            .map(ToString::to_string)
            .unwrap_or_default();
        info_span!(
            "HTTP request",
            trace_id,
            ip = %get_real_ip(request),
            method,
            path,
            user_agent,
            user_id = "None",
        )
    }

    fn on_request_end<B: MessageBody>(
        span: Span,
        outcome: &Result<ServiceResponse<B>, actix_web::Error>,
    ) {
        DefaultRootSpanBuilder::on_request_end(span, outcome);
    }
}

#[allow(clippy::type_complexity)]
fn http_handler(
    id: HyUuid,
    name: String,
) -> impl Fn(
    HttpRequest,
    Request,
    Data<PluginManager>,
    Data<Skynet>,
    Bytes,
) -> Pin<Box<dyn Future<Output = RspResult<HttpResponse<Bytes>>>>>
+ Clone
+ 'static {
    move |http_req, req, plugin_manager, skynet, body| {
        let r = plugin::Request::from_request(
            skynet.as_ref().to_owned(),
            req,
            &http_req,
            Body::Bytes(body),
        );
        let name = name.clone();
        Box::pin(async move {
            let rsp = plugin_manager
                .get(&id)
                .unwrap()
                .instance
                .as_ref()
                .unwrap()
                .on_route(&plugin_manager.reg, &name, &r)
                .await?;
            let mut ret =
                HttpResponse::new(StatusCode::from_u16(rsp.http_code)?).set_body(rsp.body);
            let mut prev = HeaderName::from_static("invalid-header");
            for (k, v) in rsp.header {
                if let Some(x) = k {
                    prev = HeaderName::from_str(&x)?;
                }
                ret.headers_mut()
                    .insert(prev.clone(), HeaderValue::from_bytes(&v)?);
            }
            Ok(ret)
        })
    }
}

#[allow(clippy::type_complexity)]
fn ws_handler(
    id: HyUuid,
    name: String,
) -> impl Fn(
    HttpRequest,
    Request,
    Data<PluginManager>,
    Data<Skynet>,
    Payload,
) -> Pin<Box<dyn Future<Output = Result<HttpResponse, actix_web::Error>>>>
+ Clone
+ 'static {
    move |http_req, req, plugin_manager, skynet, payload| {
        let trace_id = req.trace_id();
        let mut r = plugin::Request::from_request(
            skynet.as_ref().to_owned(),
            req,
            &http_req,
            Body::Message(WSMessage::Close),
        );
        let name = name.clone();
        Box::pin(async move {
            let (response, mut session, mut stream) = actix_ws::handle(&http_req, payload)?;

            let mut rx = WEBSOCKETIMPL_INSTANCE.add(trace_id, session.clone());
            spawn(async move {
                r.body = Body::Message(WSMessage::Connect);
                let _ = plugin_manager
                    .get(&id)
                    .unwrap()
                    .instance
                    .as_ref()
                    .unwrap()
                    .on_route(&plugin_manager.reg, &name, &r)
                    .await;
                loop {
                    select! {
                        _ = rx.recv() =>{
                            break;
                        },
                        msg = stream.next() => {
                            let msg = if let Some(msg) = msg {
                                msg
                            } else {
                                break;
                            };
                            let msg = match msg {
                                Ok(msg) => msg,
                                Err(e) => {
                                    debug!(error = %e, "Websocket error");
                                    continue;
                                }
                            };
                            match msg {
                                Message::Ping(bytes) => {
                                    if session.pong(&bytes).await.is_err() {
                                        break;
                                    }
                                    continue
                                }
                                Message::Text(msg) => {
                                    r.body = Body::Message(WSMessage::Text(msg));
                                }
                                Message::Binary(msg) => {
                                    r.body = Body::Message(WSMessage::Binary(msg));
                                }
                                _ => break,
                            }
                            let _ = plugin_manager
                                .get(&id)
                                .unwrap()
                                .instance
                                .as_ref()
                                .unwrap()
                                .on_route(&plugin_manager.reg, &name, &r)
                                .await;
                        },
                    };
                }
                let _ = session.close(None).await;
                WEBSOCKETIMPL_INSTANCE.remove(&trace_id);
                r.body = Body::Message(WSMessage::Close);
                let _ = plugin_manager
                    .get(&id)
                    .unwrap()
                    .instance
                    .as_ref()
                    .unwrap()
                    .on_route(&plugin_manager.reg, &name, &r)
                    .await;
            });

            Ok(response)
        })
    }
}

pub fn wrap_router(r: Vec<Router>, disable_csrf: bool) -> Vec<actix_cloud::router::Router> {
    r.into_iter()
        .map(|r| {
            let route = match r.route {
                RouterType::Inner(name) => api_call(&name, r.method.get_route()),
                RouterType::Http(id, name) => r.method.get_route().to(http_handler(id, name)),
                RouterType::Websocket(id, name) => r.method.get_route().to(ws_handler(id, name)),
            };
            actix_cloud::router::Router {
                path: r.path.clone(),
                route,
                checker: Some(Rc::new(AuthChecker::new(r.checker))),
                csrf: if disable_csrf {
                    CSRFType::Disabled
                } else {
                    r.csrf
                },
            }
        })
        .collect()
}

pub struct AuthChecker {
    perm: PermChecker,
}

impl AuthChecker {
    pub fn new(perm: PermChecker) -> Self {
        Self { perm }
    }

    async fn get_request(req: &mut ServiceRequest) -> Result<Request> {
        let s = req.get_session();
        let id = s.get::<HyUuid>("_id")?;
        let perm = match &id {
            Some(x) => {
                let db = req.app_data::<Data<DatabaseConnection>>().unwrap();
                service::get_user_perm(db.as_ref(), x).await?
            }
            None => HashMap::from([(
                GUEST_ID,
                PermissionItem {
                    name: GUEST_NAME.to_owned(),
                    pid: GUEST_ID,
                    perm: PERM_ALL,
                    ..Default::default()
                },
            )]),
        };
        let span = Span::current();
        span.record(
            "user_id",
            id.map(|x| x.to_string()).unwrap_or(String::from("None")),
        );
        Ok(Request {
            uid: id,
            username: s.get::<String>("name")?,
            perm,
            extension: req.extensions().get::<Arc<Extension>>().unwrap().to_owned(),
        })
    }
}

#[async_trait(?Send)]
impl Checker for AuthChecker {
    async fn check(&self, req: &mut ServiceRequest) -> Result<bool> {
        let r = Self::get_request(req).await?;
        let result = self.perm.check(&r.perm);
        if result {
            req.extensions_mut().insert(r);
            Ok(true)
        } else {
            Ok(false)
        }
    }
}
