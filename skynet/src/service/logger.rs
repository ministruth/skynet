use std::sync::OnceLock;

use actix_cloud::tokio::sync::mpsc::UnboundedSender;
use serde_json::{Map, Value};
use skynet_api::ffi_rpc::{
    self, async_trait, ffi_rpc_macro::plugin_impl_trait, registry::Registry, rmp_serde,
};

pub static LOGGER_INSTANCE: OnceLock<LoggerImpl> = OnceLock::new();

pub struct LoggerImpl {
    tx: Option<UnboundedSender<Map<String, Value>>>,
}

impl LoggerImpl {
    pub(super) fn new(tx: Option<UnboundedSender<Map<String, Value>>>) -> Self {
        Self { tx }
    }
}

#[plugin_impl_trait(LOGGER_INSTANCE.get().unwrap())]
impl skynet_api::service::skynet::Logger for LoggerImpl {
    async fn log(&self, _: &Registry, json: String) {
        if let Some(tx) = &self.tx {
            let _ = tx.send(serde_json::from_str(&json).unwrap());
        }
    }
}
