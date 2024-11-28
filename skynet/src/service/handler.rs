use std::sync::{atomic::Ordering, OnceLock};

use actix_cloud::{actix_web::web::Data, state::GlobalState};
use skynet_api::{
    ffi_rpc::{self, async_trait, bincode, ffi_rpc_macro::plugin_impl_trait, registry::Registry},
    HyUuid, MenuItem, Skynet,
};

pub static HANDLER_INSTANCE: OnceLock<HandlerImpl> = OnceLock::new();

pub struct HandlerImpl {
    skynet: Data<Skynet>,
    state: Data<GlobalState>,
}

impl HandlerImpl {
    pub(super) fn new(skynet: Data<Skynet>, state: Data<GlobalState>) -> Self {
        Self { skynet, state }
    }
}

#[plugin_impl_trait(HANDLER_INSTANCE.get().unwrap())]
impl skynet_api::service::skynet::Handler for HandlerImpl {
    async fn add_menu_badge(&self, _: &Registry, id: HyUuid, delta: i32) -> bool {
        fn dfs(id: HyUuid, delta: i32, item: &Vec<MenuItem>) -> bool {
            for i in item {
                if i.id == id {
                    i.badge.fetch_add(delta, Ordering::SeqCst);
                    return true;
                }
                if dfs(id, delta, &i.children) {
                    return true;
                }
            }
            false
        }
        dfs(id, delta, &self.skynet.menu)
    }

    async fn set_menu_badge(&self, _: &Registry, id: HyUuid, value: i32) -> bool {
        fn dfs(id: HyUuid, value: i32, item: &Vec<MenuItem>) -> bool {
            for i in item {
                if i.id == id {
                    i.badge.store(value, Ordering::SeqCst);
                    return true;
                }
                if dfs(id, value, &i.children) {
                    return true;
                }
            }
            false
        }
        dfs(id, value, &self.skynet.menu)
    }

    async fn stop_server(&self, _: &Registry, graceful: bool) {
        self.state.server.stop(graceful);
    }
}
