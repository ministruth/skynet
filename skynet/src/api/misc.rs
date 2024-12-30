use std::{collections::BTreeMap, net::IpAddr, sync::atomic::Ordering};

use actix_web_validator::QsQuery;
use async_recursion::async_recursion;
use maxminddb::{geoip2::Country, MaxMindDBError, Reader};
use serde::{Deserialize, Serialize};

use actix_cloud::{
    actix_web::web::Data,
    response::{JsonResponse, RspResult},
    state::GlobalState,
    t,
};
use skynet_api::{ffi_rpc::registry::Registry, plugin::Plugin, request::Request, MenuItem, Skynet};
use validator::Validate;

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

#[derive(Debug, Validate, Deserialize)]
pub struct GetGeoipReq {
    pub ip: IpAddr,
}
pub async fn geoip(
    param: QsQuery<GetGeoipReq>,
    req: Request,
    state: Data<GlobalState>,
    geoip: Data<Option<Reader<Vec<u8>>>>,
) -> RspResult<JsonResponse> {
    if param.ip.is_loopback() {
        finish_data!(t!(state.locale, "text.loopback", &req.extension.lang));
    }
    let unknown = t!(state.locale, "text.unknown", &req.extension.lang);
    match geoip.as_ref() {
        Some(geoip) => {
            let country = geoip.lookup::<Country>(param.ip);
            if let Err(e) = &country {
                if let MaxMindDBError::AddressNotFoundError(_) = e {
                    finish_data!(unknown);
                }
            }
            let country = country?;
            finish_data!(get_geoip_country(country, &req.extension.lang, unknown))
        }
        None => finish_data!(t!(state.locale, "text.na", &req.extension.lang)),
    }
}

fn get_geoip_country(country: Country, lang: &str, default: String) -> String {
    let translate = |mut s: BTreeMap<&str, &str>| -> Option<String> {
        let x = s.remove(&lang[0..2]);
        if x.is_some() {
            return x.map(Into::into);
        }
        let x = s.remove(lang);
        if x.is_some() {
            return x.map(Into::into);
        }
        return s.remove("en").map(Into::into);
    };
    if let Some(c) = country.country {
        if let Some(n) = c.names {
            if let Some(x) = translate(n) {
                return x;
            }
        }
    }
    if let Some(c) = country.represented_country {
        if let Some(n) = c.names {
            if let Some(x) = translate(n) {
                return x;
            }
        }
    }
    if let Some(c) = country.registered_country {
        if let Some(n) = c.names {
            if let Some(x) = translate(n) {
                return x;
            }
        }
    }
    return default;
}
