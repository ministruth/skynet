use actix_web::{
    web::{Data, Path},
    Responder,
};
use actix_web_validator::{Json, QsQuery};
use sea_orm::{DatabaseConnection, TransactionTrait};
use serde::Deserialize;
use serde_json::json;
use skynet::{
    finish,
    permission::PermissionItem,
    plugin::PluginStatus,
    request::{
        unique_validator, PaginationParam, Request, Response, ResponseCode, RspResult, SortType,
    },
    HyUuid, MenuItem, Skynet,
};
use std::{collections::HashMap, fs::remove_dir_all, path};
use tracing::info;
use validator::Validate;

use crate::Cli;

fn get_authorized_plugins(skynet: &Skynet, perm: &HashMap<HyUuid, PermissionItem>) -> Vec<HyUuid> {
    fn dfs(base: &[MenuItem], perm: &HashMap<HyUuid, PermissionItem>) -> Vec<HyUuid> {
        let mut ret = Vec::new();
        for i in base {
            if i.check(perm) {
                if i.path.starts_with("/plugin/") {
                    let ids: Vec<&str> = i.path.split('/').collect();
                    if ids.len() >= 3 {
                        if let Ok(x) = HyUuid::parse(ids[2]) {
                            ret.push(x);
                        }
                    }
                }
                ret.append(&mut dfs(&i.children, perm));
            }
        }
        ret
    }
    let mut ret = dfs(&skynet.menu, perm);
    ret.dedup();
    ret
}

pub async fn get_entries(req: Request, skynet: Data<Skynet>) -> RspResult<impl Responder> {
    finish!(Response::data(get_authorized_plugins(&skynet, &req.perm)));
}

#[derive(Debug, Validate, Deserialize)]
pub struct GetReq {
    #[validate(custom = "unique_validator")]
    status: Option<Vec<PluginStatus>>,
    text: Option<String>,
    priority_sort: Option<SortType>,

    #[serde(flatten)]
    #[validate]
    page: PaginationParam,
}

pub async fn get(param: QsQuery<GetReq>, skynet: Data<Skynet>) -> RspResult<impl Responder> {
    let mut data: Vec<serde_json::Value> = skynet
        .plugin
        .plugin
        .read()
        .iter()
        .filter(|p| {
            if let Some(x) = &param.status {
                if !x.contains(&p.status) {
                    return false;
                }
            }
            if let Some(x) = &param.text {
                if !p.id.to_string().contains(x) && !p.name.contains(x) {
                    return false;
                }
            }
            true
        })
        .map(|x| json!(x))
        .collect();
    if param.priority_sort.is_some_and(|x| x.is_desc()) {
        data.reverse();
    }
    finish!(Response::data(param.page.split(data)));
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutReq {
    enable: bool,
}

pub async fn put(
    id: Path<HyUuid>,
    db: Data<DatabaseConnection>,
    param: Json<PutReq>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    if !skynet.plugin.set(&tx, &skynet, &id, param.enable).await? {
        finish!(Response::not_found());
    }
    tx.commit().await?;
    info!(success=true, id = %id, enable = param.enable, "Put plugin");
    finish!(Response::ok());
}

pub async fn delete(
    id: Path<HyUuid>,
    db: Data<DatabaseConnection>,
    cli: Data<Cli>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    if let Some(x) = skynet.plugin.get(&id) {
        if x.status != PluginStatus::Unload {
            finish!(Response::new(ResponseCode::CodePluginLoaded));
        }
        let tx = db.begin().await?;
        skynet.plugin.set(&tx, &skynet, &id, false).await?;
        tx.commit().await?;
        // Ignore error from this point.
        let _ = remove_dir_all(path::Path::new(&cli.plugin).join(&x.path));
        let _ = remove_dir_all(
            path::Path::new("assets")
                .join("_plugin")
                .join(x.id.to_string()),
        );
        skynet.plugin.plugin.write().retain(|v| v.id != x.id);
    } else {
        finish!(Response::not_found());
    }
    info!(success = true, id = %id, "Delete plugin");
    finish!(Response::data(1));
}
