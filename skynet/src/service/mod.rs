use std::{cmp, collections::HashMap};

use abi_stable::{
    sabi_extern_fn,
    std_types::{RString, RVec},
};
use actix_cloud::{actix_web::web::Data, state::GlobalState};
use handler::{HANDLER_INSTANCE, HandlerImpl};
use logger::{LOGGER_INSTANCE, LoggerImpl};
use skynet_api::{
    HyUuid, Result, Skynet,
    anyhow::anyhow,
    ffi_rpc::{
        self, async_ffi::BorrowingFfiFuture, ffi_rpc_macro::plugin_impl_mock, registry::Registry,
    },
    permission::{
        GUEST_ID, GUEST_NAME, PERM_ALL, PermissionItem, ROOT_ID, ROOT_NAME, USER_ID, USER_NAME,
    },
    sea_orm::{DatabaseConnection, TransactionTrait},
    viewer::{groups::GroupViewer, permissions::PermissionViewer},
};
use webpush::{WEBPUSH_INSTANCE, WebpushImpl};
use websocket::WebsocketImpl;

use crate::webpush::WebpushManager;

pub mod handler;
pub mod logger;
pub mod webpush;
pub mod websocket;

#[plugin_impl_mock]
pub struct ServiceImpl;

pub fn init(state: &GlobalState, manager: WebpushManager) {
    let _ = LOGGER_INSTANCE.set(LoggerImpl::new(state.logger.clone().map(|x| x.sender())));
    let _ = WEBPUSH_INSTANCE.set(WebpushImpl::new(manager));
}

pub fn init_handler(skynet: Data<Skynet>, state: Data<GlobalState>) {
    let _ = HANDLER_INSTANCE.set(HandlerImpl::new(skynet, state));
}

#[sabi_extern_fn]
pub fn _ffi_call(
    func: RString,
    reg: &Registry,
    param: RVec<u8>,
) -> BorrowingFfiFuture<'_, RVec<u8>> {
    BorrowingFfiFuture::new(async move {
        if func
            .as_str()
            .starts_with("skynet_api::service::skynet::Logger::")
        {
            return LoggerImpl::parse_skynet_api_service_skynet_logger(func, reg, param).await;
        }
        if func
            .as_str()
            .starts_with("skynet_api::service::skynet::Handler::")
        {
            return HandlerImpl::parse_skynet_api_service_skynet_handler(func, reg, param).await;
        }
        if func
            .as_str()
            .starts_with("skynet_api::service::skynet::Websocket::")
        {
            return WebsocketImpl::parse_skynet_api_service_skynet_websocket(func, reg, param)
                .await;
        }
        if func
            .as_str()
            .starts_with("skynet_api::service::skynet::Webpush::")
        {
            return WebpushImpl::parse_skynet_api_service_skynet_webpush(func, reg, param).await;
        }
        panic!(
            "{}",
            format!("Function `{func}` is not defined in the library")
        );
    })
}

pub async fn get_user_perm(
    db: &DatabaseConnection,
    uid: &HyUuid,
) -> Result<HashMap<HyUuid, PermissionItem>> {
    let mut ret = get_user_dbperm(db, uid).await?;
    if uid.is_nil() {
        ret.insert(
            ROOT_ID,
            PermissionItem {
                name: ROOT_NAME.to_owned(),
                pid: ROOT_ID,
                perm: PERM_ALL,
                ..Default::default()
            },
        );
    }
    ret.insert(
        USER_ID,
        PermissionItem {
            name: USER_NAME.to_owned(),
            pid: USER_ID,
            perm: PERM_ALL,
            ..Default::default()
        },
    );
    ret.insert(
        GUEST_ID,
        PermissionItem {
            name: GUEST_NAME.to_owned(),
            pid: GUEST_ID,
            perm: PERM_ALL,
            ..Default::default()
        },
    );
    Ok(ret)
}

pub async fn get_user_dbperm(
    db: &DatabaseConnection,
    uid: &HyUuid,
) -> Result<HashMap<HyUuid, PermissionItem>> {
    let mut ret: HashMap<HyUuid, PermissionItem> = HashMap::new();
    let tx = db.begin().await?;
    let groups = GroupViewer::find_user_group(&tx, uid, false).await?;
    for i in groups {
        let perm = PermissionViewer::find_group(&tx, &i.id).await?;
        for mut j in perm {
            let origin_perm = j.perm;
            if let Some(x) = ret.remove(&j.pid) {
                j.perm |= x.perm;
                j.created_at = cmp::min(j.created_at, x.created_at);
                j.updated_at = cmp::max(j.updated_at, x.updated_at);
                j.origin = x.origin;
            }
            j.origin.push((
                j.gid.ok_or(anyhow!("GID is null"))?,
                j.ug_name.clone(),
                origin_perm,
            ));
            ret.insert(j.pid, j);
        }
    }
    let users = PermissionViewer::find_user(&tx, uid).await?;
    tx.commit().await?;
    for i in users {
        ret.insert(i.pid, i);
    }
    Ok(ret)
}
