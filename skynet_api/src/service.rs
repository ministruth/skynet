pub const SKYNET_SERVICE: &str = "skynet";

#[cfg(feature = "service-result")]
mod result {
    use std::fmt::Display;

    use anyhow::anyhow;
    use derivative::Derivative;
    use serde::{Deserialize, Serialize};

    #[derive(Serialize, Deserialize, Derivative)]
    #[derivative(Debug = "transparent")]
    #[serde(transparent)]
    pub struct SError(String);

    impl SError {
        pub fn new(str: &str) -> Self {
            Self(str.to_owned())
        }
    }

    impl<T> From<T> for SError
    where
        T: Display,
    {
        fn from(value: T) -> Self {
            SError(value.to_string())
        }
    }

    impl From<SError> for anyhow::Error {
        fn from(value: SError) -> Self {
            anyhow!("{}", value.0)
        }
    }

    pub type SResult<T> = Result<T, SError>;
}

#[cfg(feature = "service-skynet")]
pub mod skynet {
    use crate::HyUuid;
    use crate::{permission::PermChecker, plugin::WSMessage};

    use super::*;
    use ffi_rpc::{
        abi_stable, async_trait,
        ffi_rpc_macro::{plugin_api_struct, plugin_api_trait},
        rmp_serde,
    };
    use serde::{Deserialize, Serialize};

    #[plugin_api_struct]
    #[derive(Clone)]
    pub struct Service;

    #[plugin_api_trait(Service)]
    pub trait Logger: Send + Sync {
        async fn log(json: String);
    }

    #[plugin_api_trait(Service)]
    pub trait Websocket: Send + Sync {
        async fn websocket_send(id: HyUuid, msg: WSMessage) -> SResult<()>;
        async fn websocket_close(id: HyUuid);
    }

    #[plugin_api_trait(Service)]
    pub trait Handler: Send + Sync {
        async fn add_menu_badge(id: HyUuid, delta: i32) -> bool;
        async fn set_menu_badge(id: HyUuid, value: i32) -> bool;
        async fn stop_server(graceful: bool);
    }

    #[derive(Debug, Serialize, Deserialize, Default)]
    pub struct Message {
        pub title: String,
        pub body: String,
        pub url: String,
    }

    #[plugin_api_trait(Service)]
    pub trait Webpush: Send + Sync {
        async fn webpush_register(id: HyUuid, name: String, perm: PermChecker);
        async fn webpush_send(id: HyUuid, message: Message);
    }
}

#[cfg(feature = "service-result")]
pub use result::*;
#[cfg(feature = "service-skynet")]
pub use skynet::*;
