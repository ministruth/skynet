use actix_web::{
    web::{Data, Path},
    Responder,
};
use actix_web_validator::{Json, QsQuery};
use lazy_static::lazy_static;
use regex::Regex;
use sea_orm::{DatabaseConnection, TransactionTrait};
use serde::Deserialize;
use serde_json::json;
use skynet::{
    finish,
    permission::PermissionItem,
    plugin::{PluginError, PluginStatus},
    request::{
        unique_validator, PaginationParam, Request, Response, ResponseCode, RspResult, SortType,
    },
    success,
    utils::{self, parse_dataurl},
    HyUuid, MenuItem, Skynet,
};
use std::{collections::HashMap, fs::remove_dir_all, path};
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

lazy_static! {
    static ref RE_PATH_VALIDATOR: Regex = Regex::new(r"^[a-zA-Z0-9\-_]+$").unwrap();
}

#[derive(Debug, Validate, Deserialize)]
pub struct AddReq {
    #[validate(length(max = 32), regex = "RE_PATH_VALIDATOR")]
    path: String,
    file: String,
    crc32: u32,
}

pub async fn add(
    param: Json<AddReq>,
    db: Data<DatabaseConnection>,
    req: Request,
    cli: Data<Cli>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let (file, mime) = parse_dataurl(&param.file);
    if mime.is_none() || mime.unwrap().mime_type() != "application/zip" {
        finish!(Response::new(ResponseCode::CodePluginInvalid));
    }
    if crc32fast::hash(&file) != param.crc32 {
        finish!(Response::new(ResponseCode::CodePluginInvalidHash));
    }

    let dst = path::Path::new(&cli.plugin).join(&param.path);
    if dst.try_exists()? {
        finish!(Response::new(ResponseCode::CodePluginExist));
    }

    utils::unzip(&file, &dst)?;
    let tx = db.begin().await?;
    match skynet.plugin.load(&tx, &skynet, &dst).await {
        Ok(x) => {
            if !x {
                remove_dir_all(dst)?;
                finish!(Response::new(ResponseCode::CodePluginInvalid));
            }
        }
        Err(e) => {
            remove_dir_all(dst)?;
            if e.root_cause().is::<PluginError>() {
                finish!(Response::new(ResponseCode::CodePluginExist));
            }
            return Err(e.into());
        }
    }
    tx.commit().await?;
    success!(
        "Add plugin\n{}",
        json!({
            "id": "1",
            "path": param.path,
            "crc32": param.crc32,
            "ip": req.ip.ip(),
        })
    );
    finish!(Response::ok());
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutReq {
    enable: bool,
}

pub async fn put(
    id: Path<HyUuid>,
    db: Data<DatabaseConnection>,
    param: Json<PutReq>,
    req: Request,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    if !skynet.plugin.set(&tx, &skynet, &id, param.enable).await? {
        finish!(Response::not_found());
    }
    tx.commit().await?;
    success!(
        "Put plugin\n{}",
        json!({
            "id": id.as_ref(),
            "enable": param.enable,
            "ip": req.ip.ip(),
        })
    );
    finish!(Response::ok());
}

pub async fn delete(
    id: Path<HyUuid>,
    db: Data<DatabaseConnection>,
    req: Request,
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
    success!(
        "Delete plugin\n{}",
        json!({
            "id": id.as_ref(),
            "ip": req.ip.ip(),
        })
    );
    finish!(Response::data(1));
}
