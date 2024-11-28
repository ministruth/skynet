use actix_cloud::{
    actix_web::web::{Data, Path},
    response::{JsonResponse, RspResult},
    tracing::info,
};
use actix_web_validator::{Json, QsQuery};
use serde::Deserialize;
use serde_json::json;
use skynet_api::{
    finish,
    permission::PermissionItem,
    plugin::PluginStatus,
    request::{unique_validator, PaginationParam, Request, SortType},
    sea_orm::{DatabaseConnection, TransactionTrait},
    HyUuid, MenuItem, Skynet,
};
use std::{
    collections::{HashMap, HashSet},
    fs::remove_dir_all,
    path,
};
use validator::Validate;

use crate::{finish_data, finish_err, finish_ok, plugin::PluginManager, Cli, SkynetResponse};

fn get_authorized_plugins(
    skynet: &Skynet,
    perm: &HashMap<HyUuid, PermissionItem>,
) -> HashSet<HyUuid> {
    fn dfs(base: &[MenuItem], perm: &HashMap<HyUuid, PermissionItem>) -> HashSet<HyUuid> {
        let mut ret = HashSet::new();
        for i in base {
            if i.check(perm) {
                if i.path.starts_with("/plugin/") {
                    let ids: Vec<&str> = i.path.split('/').collect();
                    if ids.len() >= 3 {
                        if let Ok(x) = HyUuid::parse(ids[2]) {
                            ret.insert(x);
                        }
                    }
                }
                ret = ret
                    .union(&dfs(&i.children, perm))
                    .map(ToOwned::to_owned)
                    .collect();
            }
        }
        ret
    }
    dfs(&skynet.menu, perm)
}

pub async fn get_entries(req: Request, skynet: Data<Skynet>) -> RspResult<JsonResponse> {
    finish_data!(get_authorized_plugins(&skynet, &req.perm));
}

#[derive(Debug, Validate, Deserialize)]
pub struct GetReq {
    #[validate(custom(function = "unique_validator"))]
    pub status: Option<Vec<PluginStatus>>,
    pub text: Option<String>,
    pub priority_sort: Option<SortType>,

    #[serde(flatten)]
    #[validate(nested)]
    pub page: PaginationParam,
}

pub async fn get(param: QsQuery<GetReq>, plugin: Data<PluginManager>) -> RspResult<JsonResponse> {
    let mut data: Vec<serde_json::Value> = plugin
        .get_all()
        .into_iter()
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
    finish_data!(param.page.split(data));
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutReq {
    pub enable: bool,
}

pub async fn put(
    id: Path<HyUuid>,
    param: Json<PutReq>,
    plugin: Data<PluginManager>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    let tx = db.begin().await?;
    if !plugin.set(&tx, &id, param.enable).await? {
        finish!(JsonResponse::not_found());
    }
    tx.commit().await?;
    info!(success=true, id = %id, enable = param.enable, "Put plugin");
    finish_ok!();
}

pub async fn delete(
    id: Path<HyUuid>,
    cli: Data<Cli>,
    db: Data<DatabaseConnection>,
    plugin: Data<PluginManager>,
) -> RspResult<JsonResponse> {
    if let Some(x) = plugin.get(&id) {
        if x.status != PluginStatus::Unload {
            finish_err!(SkynetResponse::PluginLoaded);
        }
        let tx = db.begin().await?;
        plugin.set(&tx, &id, false).await?;
        tx.commit().await?;
        // Ignore error from this point.
        let _ = remove_dir_all(path::Path::new(&cli.plugin).join(&x.path));
        let _ = remove_dir_all(
            path::Path::new("assets")
                .join("_plugin")
                .join(x.id.to_string()),
        );
        plugin.delete(&x.id);
    } else {
        finish!(JsonResponse::not_found());
    }
    info!(success = true, id = %id, "Delete plugin");
    finish_data!(1);
}
