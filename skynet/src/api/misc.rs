use serde::Serialize;

use actix_cloud::{
    response::{JsonResponse, RspResult},
    state::GlobalState,
    t,
};
use skynet_api::{request::Request, MenuItem, Skynet};

use crate::{finish_data, finish_err, finish_ok, SkynetResponse};

pub async fn get_menus(req: Request) -> RspResult<JsonResponse> {
    #[derive(Serialize)]
    struct Rsp {
        name: String,
        path: String,
        icon: String,
        badge: i64,
        #[serde(skip_serializing_if = "Vec::is_empty")]
        children: Vec<Rsp>,
    }
    fn dfs(skynet: &Skynet, state: &GlobalState, req: &Request, item: &[MenuItem]) -> Vec<Rsp> {
        let mut rsp = Vec::new();
        for i in item {
            if i.check(&req.perm) {
                let mut element = Rsp {
                    name: t!(state.locale, &i.name, &req.extension.lang),
                    path: i.path.clone(),
                    icon: i.icon.clone(),
                    badge: 0,
                    children: Vec::new(),
                };
                if let Some(x) = &i.badge_func {
                    element.badge = x(skynet);
                }
                element.children = dfs(skynet, state, req, &i.children);
                // hide empty menu group
                if !(element.children.is_empty() && element.path.is_empty() && i.omit_empty) {
                    rsp.push(element);
                }
            }
        }
        rsp
    }

    finish_data!(dfs(&req.skynet, &req.state, &req, &req.skynet.menu));
}

pub async fn shutdown(req: Request) -> RspResult<JsonResponse> {
    req.state.server.stop(true);
    finish_ok!();
}

pub async fn health(req: Request) -> RspResult<JsonResponse> {
    if *req.state.server.running.read() {
        finish_ok!();
    }
    finish_err!(SkynetResponse::NotReady);
}
