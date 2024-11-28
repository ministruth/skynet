use std::{cmp, collections::HashMap, sync::OnceLock};

use abi_stable::{
    sabi_extern_fn,
    std_types::{RString, RVec},
};
use actix_cloud::{actix_web::web::Data, state::GlobalState};
use handler::{HandlerImpl, HANDLER_INSTANCE};
use logger::{LoggerImpl, LOGGER_INSTANCE};
use skynet_api::{
    anyhow::anyhow,
    ffi_rpc::{
        self, async_ffi::BorrowingFfiFuture, ffi_rpc_macro::plugin_impl_mock, registry::Registry,
    },
    permission::PermissionItem,
    sea_orm::{DatabaseConnection, TransactionTrait},
    viewer::{groups::GroupViewer, permissions::PermissionViewer},
    HyUuid, Result, Skynet,
};
use websocket::WebsocketImpl;

pub mod handler;
pub mod logger;
pub mod websocket;

pub static SERVICEIMPL_INSTANCE: OnceLock<ServiceImpl> = OnceLock::new();

#[plugin_impl_mock]
pub struct ServiceImpl {
    db: DatabaseConnection,
}

impl ServiceImpl {
    fn new(db: DatabaseConnection) -> Self {
        Self { db }
    }
}

pub fn init(state: &GlobalState, db: DatabaseConnection) {
    let _ = SERVICEIMPL_INSTANCE.set(ServiceImpl::new(db));
    let _ = LOGGER_INSTANCE.set(LoggerImpl::new(state.logger.clone().map(|x| x.sender())));
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
        panic!(
            "{}",
            format!("Function `{func}` is not defined in the library")
        );
    })
}

impl ServiceImpl {
    /// Get merged user permission.
    ///
    /// # Errors
    ///
    /// Will return `Err` when db error.
    pub async fn get_user_perm(&self, uid: &HyUuid) -> Result<Vec<PermissionItem>> {
        let mut ret: HashMap<String, PermissionItem> = HashMap::new();
        let tx = self.db.begin().await?;
        let groups = GroupViewer::find_user_group(&tx, uid, false).await?;
        for i in groups {
            let perm = PermissionViewer::find_group(&tx, &i.id).await?;
            for mut j in perm {
                let origin_perm = j.perm;
                if let Some(x) = ret.remove(&j.name) {
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
                ret.insert(j.name.clone(), j);
            }
        }
        let users = PermissionViewer::find_user(&tx, uid).await?;
        tx.commit().await?;
        for i in users {
            ret.insert(i.name.clone(), i);
        }
        Ok(ret.into_values().collect())
    }
}
