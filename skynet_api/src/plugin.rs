use bytes::Bytes;
use bytestring::ByteString;
use enum_as_inner::EnumAsInner;
use serde::{Deserialize, Serialize};
use serde_repr::{Deserialize_repr, Serialize_repr};

#[derive(
    Serialize_repr, Deserialize_repr, Debug, Clone, Copy, PartialEq, Eq, Hash, EnumAsInner,
)]
#[repr(u8)]
pub enum PluginStatus {
    Unload = 0,
    PendingDisable,
    PendingEnable,
    Enable,
}

#[derive(thiserror::Error, Debug)]
pub enum PluginError {
    #[error("No such route")]
    NoSuchRoute,

    #[error("Session not found")]
    SessionNotFound,

    #[error("Session is closed")]
    SessionClosed,

    #[error("{0}")]
    Custom(String),
}

#[derive(Serialize, Deserialize, Clone, Debug)]
pub enum WSMessage {
    Connect,
    Text(ByteString),
    Binary(Bytes),
    Close,
}

#[derive(Serialize, Deserialize, Clone)]
pub enum Body {
    Bytes(Bytes),
    Message(WSMessage),
}

#[cfg(feature = "plugin-api")]
pub mod api {
    use crate::{Skynet, request::Router, service::SResult};

    use super::*;
    use std::path::PathBuf;

    use ffi_rpc::{
        abi_stable, async_trait,
        ffi_rpc_macro::{self, plugin_api},
        rmp_serde,
    };

    /// Plugin interface, all plugins should implement this trait.
    ///
    /// # Lifecycle
    ///
    /// - Skynet init(db, redis, etc.)
    /// - Check plugin enabled
    /// - **<`on_load`>**
    /// - **<`on_register`>**
    /// - Skynet running (**<`on_route`>****)
    /// - ...
    /// - **<`on_unload`>**
    /// - Skynet shutdown
    #[plugin_api(Plugin)]
    pub trait PluginApi: Send + Sync {
        /// Fired when the plugin is loaded.
        ///
        /// Basic implementation:
        /// ```ignore
        /// async fn on_load(&self, _: &Registry, skynet: Skynet, _runtime_path: PathBuf) -> SResult<Skynet> {
        ///     Ok(skynet)
        /// }
        /// ```
        async fn on_load(skynet: Skynet, runtime_path: PathBuf) -> SResult<Skynet>;

        /// Fired when register routers.
        ///
        /// Basic implementation:
        /// ```ignore
        /// async fn on_register(&self, _: &Registry, _skynet: Skynet, r: Vec<Router>) -> Vec<Router> {
        ///     r
        /// }
        /// ```
        async fn on_register(skynet: Skynet, r: Vec<Router>) -> Vec<Router>;

        /// Fired when incoming request.
        ///
        /// Basic implementation:
        /// ```ignore
        /// async fn on_route(&self, _: &Registry, _name: String, _req: Request) -> SResult<Response> {
        ///     Err(PluginError::NoSuchRoute.into())
        /// }
        /// ```
        async fn on_route(name: String, req: Request) -> SResult<Response>;

        /// Fired when the plugin is unloaded.
        ///
        /// Basic implementation:
        /// ```ignore
        /// async fn on_unload(&self, _: &Registry, _status: PluginStatus) {}
        /// ```
        async fn on_unload(status: PluginStatus);

        /// Fired when translating strings.
        ///
        /// Basic implementation:
        /// ```ignore
        /// async fn on_translate(&self, _: &Registry, str: String, _lang: String) -> String {
        ///     str
        /// }
        /// ```
        async fn on_translate(str: String, lang: String) -> String;
    }
}

#[cfg(feature = "plugin-request")]
mod request {
    use std::{
        fmt::Debug,
        future::{Ready, ready},
        net::SocketAddr,
        str::FromStr,
    };

    use actix_cloud::{
        actix_web::{
            self, FromRequest, Handler, HttpMessage, HttpRequest, Responder,
            body::to_bytes,
            dev::{Payload, ServiceRequest, ServiceResponse},
            http::header::{HeaderName, HeaderValue},
            test,
            web::Data,
        },
        chrono::{DateTime, Utc},
        state::GlobalState,
    };
    use ahash::AHashMap;
    use anyhow::{Result, anyhow};
    use ffi_rpc::registry::Registry;

    use crate::{Skynet, request::Method, service::SResult};

    use super::*;

    impl FromRequest for WSMessage {
        type Error = actix_web::Error;
        type Future = Ready<Result<WSMessage, actix_web::Error>>;

        #[inline]
        fn from_request(req: &HttpRequest, _: &mut Payload) -> Self::Future {
            ready(
                req.extensions_mut()
                    .remove::<Self>()
                    .ok_or(actix_web::error::ErrorBadGateway(anyhow!(
                        "Message is not parsed"
                    ))),
            )
        }
    }

    #[derive(Serialize, Deserialize, Clone)]
    pub struct ServerHandle {
        pub running: bool,
        pub start_time: DateTime<Utc>,
        pub stop_time: Option<DateTime<Utc>>,
    }

    #[derive(Serialize, Deserialize, Clone)]
    pub struct Request {
        pub header: AHashMap<String, Vec<u8>>,
        pub path_param: AHashMap<String, String>,
        pub method: Method,
        pub uri: String,
        pub address: SocketAddr,
        pub body: Body,

        pub skynet: Skynet,
        pub req: crate::request::Request,
        pub server_handle: ServerHandle,
    }

    impl Request {
        pub fn from_request(
            skynet: Skynet,
            req: crate::request::Request,
            http_req: &HttpRequest,
            body: Body,
        ) -> Self {
            let state = http_req.app_data::<Data<GlobalState>>().unwrap();
            Self {
                header: http_req
                    .headers()
                    .iter()
                    .map(|(k, v)| (k.to_string(), v.as_bytes().to_owned()))
                    .collect(),
                uri: http_req.uri().to_string(),
                address: req.extension.real_ip,
                body,
                method: http_req.method().to_owned().into(),
                path_param: http_req
                    .match_info()
                    .iter()
                    .map(|(x, y)| (x.to_owned(), y.to_owned()))
                    .collect(),
                req,
                skynet,
                server_handle: ServerHandle {
                    running: *state.server.running.read(),
                    start_time: *state.server.start_time.read(),
                    stop_time: *state.server.stop_time.read(),
                },
            }
        }

        pub fn into_srv_request(
            self,
            reg: Registry,
            state: Data<GlobalState>,
        ) -> Result<ServiceRequest> {
            *state.server.running.write() = self.server_handle.running;
            *state.server.start_time.write() = self.server_handle.start_time;
            *state.server.stop_time.write() = self.server_handle.stop_time;
            let mut r = test::TestRequest::default()
                .method(self.method.into())
                .uri(&self.uri)
                .peer_addr(self.address)
                .app_data(Data::new(self.skynet))
                .app_data(Data::new(reg))
                .app_data(state);
            for header in self.header {
                r = r.append_header((
                    HeaderName::from_str(&header.0)?,
                    HeaderValue::from_bytes(&header.1)?,
                ));
            }
            for param in self.path_param {
                r = r.param(param.0, param.1);
            }
            let msg = match self.body {
                Body::Bytes(bytes) => {
                    r = r.set_payload(bytes);
                    None
                }
                Body::Message(msg) => Some(msg),
            };
            let r = r.to_srv_request();
            r.extensions_mut().insert(self.req.extension.clone());
            r.extensions_mut().insert(self.req);
            if let Some(msg) = msg {
                r.extensions_mut().insert(msg);
            }
            Ok(r)
        }
    }

    #[derive(Serialize, Deserialize)]
    pub struct Response {
        pub header: AHashMap<Option<String>, Vec<u8>>,
        pub http_code: u16,
        pub body: Bytes,
    }

    impl Response {
        pub async fn from_srv_response(mut r: ServiceResponse) -> SResult<Self> {
            let rsp = r.response_mut();
            if let Some(err) = rsp.error() {
                return Err(err.into());
            }

            Ok(Response {
                header: rsp
                    .headers_mut()
                    .drain()
                    .map(|(k, v)| (k.map(|x| x.to_string()), v.as_bytes().to_owned()))
                    .collect(),
                http_code: rsp.status().into(),
                body: to_bytes(r.into_body()).await?,
            })
        }
    }

    pub async fn api_call<F, Args>(req: ServiceRequest, handler: F) -> Result<ServiceResponse>
    where
        F: Handler<Args>,
        Args: FromRequest,
        Args::Error: Debug,
        F::Output: Responder,
    {
        let (req, mut payload) = req.into_parts();
        let param = Args::from_request(&req, &mut payload)
            .await
            .map_err(|e| anyhow!("{:?}", e))?;
        let res = handler
            .call(param)
            .await
            .respond_to(&req)
            .map_into_boxed_body();

        Ok(ServiceResponse::new(req, res))
    }

    #[macro_export]
    macro_rules! route {
        {$reg:expr, $state:expr, $name:expr, $req:expr, $($key:expr => $value:path),+ $(,)?} => {{
            let reg = $reg.to_owned();
            let state = $state.to_owned();
            tokio::runtime::Handle::current().spawn_blocking(|| {
                tokio::runtime::Handle::current().block_on(async {
                    use std::str::FromStr;
                    use skynet_api::tracing::Instrument;

                    let span = actix_cloud::tracing::info_span!(
                        "HTTP request",
                        trace_id = %$req.req.trace_id(),
                        ip = %$req.req.extension.real_ip,
                        method = %$req.method,
                        path = $req
                            .uri
                            .parse::<actix_cloud::actix_web::http::Uri>()?
                            .path_and_query()
                            .map(ToString::to_string)
                            .unwrap_or_default(),
                        user_agent = $req
                            .header
                            .get(actix_cloud::actix_web::http::header::HeaderName::from_str("User-Agent").unwrap().as_str())
                            .map_or(String::new(), |x| String::from_utf8_lossy(x).to_string()),
                    );
                    async move {
                        let mut r = $req.into_srv_request(reg, state)?;
                        let ret = match $name.as_str() {
                            $(
                                $key => skynet_api::plugin::api_call(r, $value).await,
                            )+
                            _ => Err(skynet_api::plugin::PluginError::NoSuchRoute.into()),
                        }?;
                        return Response::from_srv_response(ret).await;
                    }
                    .instrument(span)
                    .await
                })
            }).await?
        }};
    }
}

#[cfg(feature = "plugin-api")]
pub use api::*;
#[cfg(feature = "plugin-request")]
pub use request::*;
