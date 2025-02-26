use crate::{
    HyUuid,
    config::Config,
    logger::Logger,
    permission::{IDTypes, PermChecker, PermissionItem},
};
use dashmap::DashMap;
use derivative::Derivative;
use enum_map::EnumMap;
use serde::{Deserialize, Serialize};
use std::{
    cmp,
    collections::HashMap,
    sync::{Arc, atomic::AtomicI32},
};

/// Main entrance providing skynet function.
#[derive(Serialize, Deserialize, Clone)]
pub struct Skynet {
    pub logger: Logger,
    pub config: Config,
    pub default_id: EnumMap<IDTypes, HyUuid>,
    pub menu: Vec<MenuItem>,
    pub warning: Arc<DashMap<String, String>>,
}

impl Skynet {
    pub fn reset_menu_badge(&mut self, id: HyUuid, badge: Arc<AtomicI32>) -> bool {
        fn dfs(id: HyUuid, badge: Arc<AtomicI32>, item: &mut Vec<MenuItem>) -> bool {
            for i in item {
                if i.id == id {
                    i.badge = badge.clone();
                    return true;
                }
                if dfs(id, badge.clone(), &mut i.children) {
                    return true;
                }
            }
            false
        }
        dfs(id, badge, &mut self.menu)
    }

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

    #[cfg(feature = "database")]
    pub async fn get_db(&self) -> anyhow::Result<sea_orm::DatabaseConnection> {
        let mut opt = sea_orm::ConnectOptions::new(&self.config.database.dsn);
        opt.sqlx_logging(false);
        sea_orm::Database::connect(opt).await.map_err(Into::into)
    }
}

#[derive(Derivative, Serialize, Deserialize, Clone)]
#[derivative(Default(new = "true"), Debug)]
pub struct MenuItem {
    pub id: HyUuid,
    pub plugin: Option<HyUuid>,
    pub name: String,
    pub path: String,
    pub icon: String,
    pub children: Vec<MenuItem>,
    pub badge: Arc<AtomicI32>,
    pub omit_empty: bool,
    pub checker: PermChecker,
}

impl MenuItem {
    pub fn check(&self, p: &HashMap<HyUuid, PermissionItem>) -> bool {
        self.checker.check(p)
    }
}
