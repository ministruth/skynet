use std::sync::OnceLock;

use skynet_api::{
    HyUuid,
    ffi_rpc::{self, async_trait, ffi_rpc_macro::plugin_impl_trait, registry::Registry, rmp_serde},
    permission::PermChecker,
    service::Message,
};

use crate::webpush::WebpushManager;

pub static WEBPUSH_INSTANCE: OnceLock<WebpushImpl> = OnceLock::new();

pub struct WebpushImpl {
    manager: WebpushManager,
}

impl WebpushImpl {
    pub(super) fn new(manager: WebpushManager) -> Self {
        Self { manager }
    }
}

#[plugin_impl_trait(WEBPUSH_INSTANCE.get().unwrap())]
impl skynet_api::service::skynet::Webpush for WebpushImpl {
    async fn webpush_register(&self, _: &Registry, id: HyUuid, name: String, perm: PermChecker) {
        self.manager.add_topic(id, name, perm).await;
    }

    async fn webpush_send(&self, _: &Registry, id: HyUuid, message: Message) {
        self.manager.send(id, message);
    }
}
