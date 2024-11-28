#[cfg(feature = "config")]
pub mod config;
#[cfg(feature = "database")]
pub mod entity;
pub mod hyuuid;
#[cfg(feature = "logger")]
pub mod logger;
pub mod permission;
#[cfg(feature = "plugin-basic")]
pub mod plugin;
pub mod request;
#[cfg(feature = "serde")]
pub mod serializer;
pub mod service;
#[cfg(feature = "skynet")]
pub mod skynet;
pub mod utils;
#[cfg(feature = "viewer")]
pub mod viewer;

#[cfg(feature = "logger")]
pub use actix_cloud::tracing;
pub use anyhow;
pub use anyhow::bail;
pub use anyhow::Result;
#[cfg(any(feature = "plugin-request", feature = "service-skynet"))]
pub use ffi_rpc;
pub use hyuuid::HyUuid;
#[cfg(feature = "request-param")]
pub use paste;
#[cfg(any(feature = "database", feature = "request-pagination"))]
pub use sea_orm;
#[cfg(feature = "skynet")]
pub use skynet::*;
pub use uuid::uuid;

pub const VERSION: &str = env!("CARGO_PKG_VERSION");
