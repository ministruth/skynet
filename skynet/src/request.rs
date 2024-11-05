use std::{
    collections::HashMap, future::Future, net::SocketAddr, pin::Pin, rc::Rc, sync::Arc,
    time::Duration,
};

use actix_cloud::{
    actix_web::{
        self,
        body::MessageBody,
        dev::{ServiceRequest, ServiceResponse},
        http::{Method, StatusCode},
        middleware::Next,
        web::{delete, get, head, patch, post, put, trace, Data, Payload},
        HttpMessage, HttpRequest,
    },
    async_trait,
    chrono::Utc,
    request::Extension,
    response::RspResult,
    router::Checker,
    session::{Session, SessionExt},
    state::GlobalState,
    tracing::{error, info, info_span, Span},
    tracing_actix_web::{DefaultRootSpanBuilder, RequestId, RootSpanBuilder},
    utils,
};
use derivative::Derivative;
use skynet_api::{
    anyhow,
    permission::{
        PermType, PermissionItem, GUEST_ID, GUEST_NAME, PERM_ALL, ROOT_ID, ROOT_NAME, USER_ID,
        USER_NAME,
    },
    request::{Request, Router, RouterType},
    sea_orm::{DatabaseConnection, TransactionTrait},
    HyUuid, Result, Skynet,
};

use crate::Cli;

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
        skynet_api::finish!(actix_cloud::response::JsonResponse::new(
            $crate::SkynetResponse::Success
        )
        .json($rsp))
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
            if !cli.verbose && !rsp.status().is_success() {
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
            user_agent
        )
    }

    fn on_request_end<B: MessageBody>(
        span: Span,
        outcome: &Result<ServiceResponse<B>, actix_web::Error>,
    ) {
        DefaultRootSpanBuilder::on_request_end(span, outcome);
    }
}

fn route_handler<F, T>(
    func: Rc<F>,
) -> impl Fn(HttpRequest, Payload) -> Pin<Box<dyn Future<Output = RspResult<T>>>> + Clone + 'static
where
    F: Fn(Request, Payload) -> Pin<Box<dyn Future<Output = RspResult<T>>>> + ?Sized + 'static,
{
    move |req, payload| {
        let func = func.clone();
        let r = req
            .extensions_mut()
            .remove::<Request>()
            .ok_or(anyhow!("Request is not parsed"))
            .unwrap();
        Box::pin(async move { func(r, payload).await })
    }
}

pub fn wrap_router(r: Vec<Router>) -> Vec<actix_cloud::router::Router> {
    r.into_iter()
        .map(|r| {
            let method = match r.method {
                Method::DELETE => delete(),
                Method::GET => get(),
                Method::HEAD => head(),
                Method::PATCH => patch(),
                Method::POST => post(),
                Method::PUT => put(),
                Method::TRACE => trace(),
                _ => unimplemented!(),
            };
            let route = match r.route {
                RouterType::Json(rt) => method.to(route_handler(rt)),
                RouterType::Http(rt) => method.to(route_handler(rt)),
                RouterType::Raw(rt) => rt,
            };
            actix_cloud::router::Router {
                path: r.path.clone(),
                route,
                checker: Some(Rc::new(AuthChecker::new(r.checker))),
                csrf: r.csrf,
            }
        })
        .collect()
}

pub struct AuthChecker {
    perm: PermType,
}

impl AuthChecker {
    pub fn new(perm: PermType) -> Self {
        Self { perm }
    }

    async fn get_request(req: &mut ServiceRequest) -> Result<Request> {
        let s = req.get_session();
        let id = s.get::<HyUuid>("_id")?;
        let mut perm = match &id {
            Some(x) => {
                let mut perm = if x.is_nil() {
                    // root
                    HashMap::from([(
                        ROOT_ID,
                        PermissionItem {
                            name: ROOT_NAME.to_owned(),
                            pid: ROOT_ID,
                            perm: PERM_ALL,
                            ..Default::default()
                        },
                    )])
                } else {
                    // user
                    let skynet = req.app_data::<Data<Skynet>>().unwrap();
                    let db = req.app_data::<Data<DatabaseConnection>>().unwrap();
                    let tx = db.begin().await.unwrap();
                    let perm = skynet
                        .get_user_perm(&tx, x)
                        .await
                        .unwrap()
                        .into_iter()
                        .map(|x| (x.pid, x))
                        .collect();
                    tx.commit().await.unwrap();
                    perm
                };
                perm.insert(
                    USER_ID,
                    PermissionItem {
                        name: USER_NAME.to_owned(),
                        pid: USER_ID,
                        perm: PERM_ALL,
                        ..Default::default()
                    },
                );
                perm
            }
            None => HashMap::new(),
        };
        perm.insert(
            GUEST_ID,
            PermissionItem {
                name: GUEST_NAME.to_owned(),
                pid: GUEST_ID,
                perm: PERM_ALL,
                ..Default::default()
            },
        );
        let session = req.extract::<Session>().await.unwrap();
        Ok(Request {
            uid: id,
            username: s.get::<String>("name")?,
            perm,
            extension: req.extensions().get::<Arc<Extension>>().unwrap().to_owned(),
            skynet: req.app_data::<Data<Skynet>>().unwrap().to_owned(),
            state: req.app_data::<Data<GlobalState>>().unwrap().to_owned(),
            session,
            http_request: req.request().clone(),
        })
    }
}

#[async_trait(?Send)]
impl Checker for AuthChecker {
    async fn check(&self, req: &mut ServiceRequest) -> Result<bool> {
        let r = Self::get_request(req).await?;
        let result = match &self.perm {
            PermType::Entry(x) => x.check(&r.perm),
            PermType::Custom(x) => x(&r.perm),
        };
        if result {
            req.extensions_mut().insert(r);
            Ok(true)
        } else {
            Ok(false)
        }
    }
}
