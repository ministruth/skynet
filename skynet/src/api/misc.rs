use std::sync::atomic::Ordering;

use async_recursion::async_recursion;
use serde::Serialize;

use actix_cloud::{
    actix_web::web::Data,
    response::{JsonResponse, RspResult},
    state::GlobalState,
    t,
};
use skynet_api::{ffi_rpc::registry::Registry, plugin::Plugin, request::Request, MenuItem, Skynet};

use crate::{finish_data, finish_err, finish_ok, plugin::PluginManager, SkynetResponse};

pub async fn get_menus(
    req: Request,
    plugin: Data<PluginManager>,
    state: Data<GlobalState>,
    skynet: Data<Skynet>,
) -> RspResult<JsonResponse> {
    #[derive(Serialize)]
    struct Rsp {
        name: String,
        path: String,
        icon: String,
        badge: i32,
        #[serde(skip_serializing_if = "Vec::is_empty")]
        children: Vec<Rsp>,
    }
    #[async_recursion(?Send)]
    async fn dfs(
        reg: &Registry,
        req: &Request,
        state: &GlobalState,
        item: &[MenuItem],
    ) -> Vec<Rsp> {
        let mut rsp = Vec::new();
        for i in item {
            if i.check(&req.perm) {
                let name = match i.plugin {
                    Some(id) => {
                        Plugin::from(reg.get(&id.to_string()).unwrap())
                            .on_translate(reg, &i.name, &req.extension.lang)
                            .await
                    }
                    None => t!(state.locale, &i.name, &req.extension.lang),
                };
                let mut element = Rsp {
                    name,
                    path: i.path.clone(),
                    icon: i.icon.clone(),
                    badge: i.badge.load(Ordering::Relaxed),
                    children: Vec::new(),
                };
                element.children = dfs(reg, req, state, &i.children).await;
                // hide empty menu group
                if !(element.children.is_empty() && element.path.is_empty() && i.omit_empty) {
                    rsp.push(element);
                }
            }
        }
        rsp
    }

    finish_data!(dfs(&plugin.reg, &req, &state, &skynet.menu).await);
}

pub async fn shutdown(state: Data<GlobalState>) -> RspResult<JsonResponse> {
    state.server.stop(true);
    finish_ok!();
}

pub async fn health(state: Data<GlobalState>) -> RspResult<JsonResponse> {
    if *state.server.running.read() {
        finish_ok!();
    }
    finish_err!(SkynetResponse::NotReady);
}
