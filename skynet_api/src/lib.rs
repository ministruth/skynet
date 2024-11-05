#[cfg(feature = "api")]
pub mod api;
#[cfg(feature = "config")]
pub mod config;
#[cfg(feature = "database")]
pub mod entity;
#[cfg(feature = "skynet")]
pub mod handler;
pub mod hyuuid;
#[cfg(feature = "logger")]
pub mod logger;
#[cfg(feature = "permission")]
pub mod permission;
#[cfg(feature = "plugin")]
pub mod plugin;
#[cfg(feature = "skynet")]
pub mod request;
#[cfg(feature = "serde")]
pub mod serializer;
#[cfg(feature = "skynet")]
pub mod skynet;
pub mod utils;

pub use anyhow::anyhow;
pub use anyhow::bail;
pub use anyhow::Result;
pub use hyuuid::HyUuid;
#[cfg(feature = "skynet")]
pub use paste;
#[cfg(feature = "database")]
pub use sea_orm;
#[cfg(feature = "skynet")]
pub use skynet::*;
pub use uuid::uuid;

pub const VERSION: &str = env!("CARGO_PKG_VERSION");
