use std::{future::Future, net::SocketAddr, pin::Pin, rc::Rc, time::Duration};

use derivative::Derivative;
use skynet_api::{
    actix_cloud::{
        actix_web::{
            self,
            body::MessageBody,
            dev::{ServiceRequest, ServiceResponse},
            http::StatusCode,
            middleware::Next,
            web::Data,
            HttpMessage, HttpRequest, HttpResponse,
        },
        chrono::Utc,
        state::GlobalState,
        tracing_actix_web::{DefaultRootSpanBuilder, RequestId, RootSpanBuilder},
        utils,
    },
    tracing::{error, info, info_span, Span},
    Result, Skynet,
};

pub const CSRF_COOKIE: &str = "CSRF_TOKEN";
pub const CSRF_HEADER: &str = "X-CSRF-Token";
pub const TRACE_HEADER: &str = "x-trace-id"; // This should be lowercase.

#[macro_export]
macro_rules! finish_ok {
    () => {
        skynet_api::finish!(skynet_api::actix_cloud::response::JsonResponse::new(
            $crate::SkynetResponse::Success
        ))
    };
}

#[macro_export]
macro_rules! finish_err {
    ($rsp:path) => {
        skynet_api::finish!(skynet_api::actix_cloud::response::JsonResponse::new($rsp))
    };
}

#[macro_export]
macro_rules! finish_data {
    ($rsp:expr) => {
        skynet_api::finish!(skynet_api::actix_cloud::response::JsonResponse::new(
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
            Ok(match rsp.status() {
                StatusCode::BAD_REQUEST => {
                    rsp.into_response(HttpResponse::BadRequest().finish().map_into_right_body())
                }
                StatusCode::NOT_FOUND => {
                    rsp.into_response(HttpResponse::NotFound().finish().map_into_right_body())
                }
                StatusCode::INTERNAL_SERVER_ERROR => rsp.into_response(
                    HttpResponse::InternalServerError()
                        .finish()
                        .map_into_right_body(),
                ),
                _ => rsp.map_into_left_body(),
            })
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
            Err(match rsp.status() {
                StatusCode::BAD_REQUEST => actix_web::error::ErrorBadRequest(""),
                StatusCode::NOT_FOUND => actix_web::error::ErrorNotFound(""),
                StatusCode::INTERNAL_SERVER_ERROR => actix_web::error::ErrorInternalServerError(""),
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
