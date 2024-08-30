use std::{cmp, collections::HashMap};

pub use actix_cloud;
pub use actix_cloud::{anyhow::anyhow, async_trait, tracing, Result};
pub use hyuuid::HyUuid;
pub use parking_lot;
pub use paste;
pub use sea_orm;
pub use uuid::uuid;

use api::APIManager;
use config::Config;
use enum_map::EnumMap;
use handler::*;
use logger::Logger;
use permission::{IDTypes, PermissionItem};
use plugin::PluginManager;
use request::MenuItem;
use sea_orm::DatabaseTransaction;

pub mod api;
pub mod config;
pub mod entity;
pub mod handler;
pub mod hyuuid;
pub mod logger;
pub mod permission;
pub mod plugin;
pub mod request;
pub mod serializer;
pub mod utils;

pub const VERSION: &str = env!("CARGO_PKG_VERSION");

/// Main entrance providing skynet function.
pub struct Skynet {
    pub user: Box<dyn UserHandler>,
    pub group: Box<dyn GroupHandler>,
    pub perm: Box<dyn PermHandler>,
    pub notification: Box<dyn NotificationHandler>,
    pub setting: Box<dyn SettingHandler>,
    pub logger: Logger,

    pub default_id: EnumMap<IDTypes, HyUuid>,
    pub config: Config,

    pub menu: Vec<MenuItem>,
    pub plugin: PluginManager,

    pub shared_api: APIManager,
}

impl Drop for Skynet {
    fn drop(&mut self) {
        // clear API first otherwise SIGSEGV.
        self.shared_api.clear();
    }
}

impl Skynet {
    pub fn insert_menu(&mut self, item: MenuItem, pos: usize, parent: Option<HyUuid>) -> bool {
        if let Some(parent) = parent {
            let mut parent: Vec<&mut MenuItem> =
                self.menu.iter_mut().filter(|x| x.id == parent).collect();
            let Some(parent) = parent.pop() else {
                return false;
            };
            let pos = cmp::min(pos, parent.children.len());
            parent.children.insert(pos, item);
        } else {
            let pos = cmp::min(pos, self.menu.len());
            self.menu.insert(pos, item);
        }
        true
    }

    /// Get merged user permission.
    ///
    /// # Errors
    ///
    /// Will return `Err` when db error.
    pub async fn get_user_perm(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
    ) -> Result<Vec<PermissionItem>> {
        let mut ret: HashMap<String, PermissionItem> = HashMap::new();
        let groups = self.group.find_user_group(db, uid, false).await?;
        for i in groups {
            let perm = self.perm.find_group(db, &i.id).await?;
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
        let users = self.perm.find_user(db, uid).await?;
        for i in users {
            ret.insert(i.name.clone(), i);
        }
        Ok(ret.into_values().collect())
    }
}
