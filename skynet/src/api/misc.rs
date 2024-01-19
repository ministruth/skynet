use actix_web::{web::Data, Responder};
use serde::Serialize;
use skynet::{
    finish,
    request::{Request, Response, ResponseCode, RspResult},
    t, MenuItem, Skynet,
};

use crate::cmd::run::StopHandle;

pub async fn get_menus(skynet: Data<Skynet>, req: Request) -> RspResult<impl Responder> {
    #[derive(Serialize)]
    struct Rsp {
        name: String,
        path: String,
        icon: String,
        badge: i64,
        #[serde(skip_serializing_if = "Vec::is_empty")]
        children: Vec<Rsp>,
    }
    fn dfs(skynet: &Data<Skynet>, req: &Request, item: &[MenuItem]) -> Vec<Rsp> {
        let mut rsp = Vec::new();
        for i in item {
            if i.check(&req.perm) {
                let mut element = Rsp {
                    name: t!(skynet, &i.name, &req.lang),
                    path: i.path.clone(),
                    icon: i.icon.clone(),
                    badge: 0,
                    children: Vec::new(),
                };
                if let Some(x) = &i.badge_func {
                    element.badge = x(skynet);
                }
                element.children = dfs(skynet, req, &i.children);
                // hide empty menu group
                if !(element.children.is_empty() && element.path.is_empty() && i.omit_empty) {
                    rsp.push(element);
                }
            }
        }
        rsp
    }

    finish!(Response::data(dfs(&skynet, &req, &skynet.menu)));
}

pub async fn shutdown(
    skynet: Data<Skynet>,
    stop_handle: Data<StopHandle>,
) -> RspResult<impl Responder> {
    *skynet.running.write() = false;
    stop_handle.stop(true);
    finish!(Response::ok());
}

pub async fn ping(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    if *skynet.running.read() {
        finish!(Response::ok());
    }
    finish!(Response::new(ResponseCode::CodeNotReady));
}
