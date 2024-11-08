use derivative::Derivative;
use enum_map::Enum;
#[cfg(feature = "database")]
use sea_orm::FromQueryResult;
use std::{collections::HashMap, fmt};

use crate::HyUuid;

#[derive(Debug, Enum)]
pub enum IDTypes {
    /// manage user
    PermManageUserID,
    /// manage notification
    PermManageNotificationID,
    /// manage system
    PermManageSystemID,
    /// manage plugin
    PermManagePluginID,
}

impl fmt::Display for IDTypes {
    fn fmt(&self, fmt: &mut fmt::Formatter) -> fmt::Result {
        let name = match self {
            Self::PermManageUserID => "manage.user",
            Self::PermManageNotificationID => "manage.notification",
            Self::PermManageSystemID => "manage.system",
            Self::PermManagePluginID => "manage.plugin",
        };
        fmt.write_str(name)
    }
}

pub type UserPerm = i32;
/// revoke permission.
pub const PERM_REVOKE: UserPerm = -1;
/// forbid permission, alias of `PERM_NONE`.
pub const PERM_FORBIDDEN: UserPerm = 0;
/// default, no permission.
pub const PERM_NONE: UserPerm = 0;
/// write data.
pub const PERM_WRITE: UserPerm = 1;
/// read data.
pub const PERM_READ: UserPerm = 1 << 1;
/// all permission.
pub const PERM_ALL: UserPerm = (1 << 2) - 1;

/// root permission name.
pub const ROOT_NAME: &str = "root";
/// guest permission name.
pub const GUEST_NAME: &str = "guest";
/// user permission name.
pub const USER_NAME: &str = "user";

/// root permission id.
pub const ROOT_ID: HyUuid = HyUuid::nil();
/// guest permission id.
pub const GUEST_ID: HyUuid = HyUuid(uuid::uuid!("1a2d05da-a256-475c-a2b0-dd0aa1b36b4f"));
/// user permission id.
pub const USER_ID: HyUuid = HyUuid(uuid::uuid!("61ee97f9-0a4b-4215-a9c7-ace22708bb6c"));

#[derive(Derivative, Clone)]
#[derivative(Default(new = "true"), Debug)]
pub struct PermEntry {
    pub pid: HyUuid,
    pub perm: UserPerm,
}

impl PermEntry {
    pub fn check(&self, p: &HashMap<HyUuid, PermissionItem>) -> bool {
        // root
        if p.contains_key(&ROOT_ID) {
            return true;
        }
        p.get(&self.pid)
            .map_or(false, |x| (x.perm & self.perm) == self.perm)
    }

    pub fn is_guest(&self) -> bool {
        self.pid == GUEST_ID
    }

    pub fn is_user(&self) -> bool {
        self.pid == USER_ID
    }

    pub fn is_root(&self) -> bool {
        self.pid == ROOT_ID
    }

    pub const fn new_guest() -> Self {
        Self {
            pid: GUEST_ID,
            perm: PERM_ALL,
        }
    }

    pub const fn new_user() -> Self {
        Self {
            pid: USER_ID,
            perm: PERM_ALL,
        }
    }

    pub const fn new_root() -> Self {
        Self {
            pid: ROOT_ID,
            perm: PERM_ALL,
        }
    }
}

#[cfg_attr(feature = "database", derive(FromQueryResult))]
#[derive(Debug, Derivative)]
#[derivative(Default(new = "true"))]
pub struct PermissionItem {
    pub id: HyUuid,
    pub name: String,
    pub note: String,
    pub pid: HyUuid,
    pub perm: UserPerm,
    pub uid: Option<HyUuid>,
    pub gid: Option<HyUuid>,
    pub ug_name: String,
    #[cfg_attr(feature = "database", sea_orm(skip))]
    pub origin: Vec<(HyUuid, String, i32)>,
    pub created_at: i64,
    pub updated_at: i64,
}

impl From<PermissionItem> for PermEntry {
    fn from(value: PermissionItem) -> Self {
        Self {
            pid: value.pid,
            perm: value.perm,
        }
    }
}

pub type PermChecker = dyn Fn(&HashMap<HyUuid, PermissionItem>) -> bool;

pub enum PermType {
    Entry(PermEntry),
    Custom(Box<PermChecker>),
}
