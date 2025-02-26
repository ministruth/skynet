use crate::HyUuid;

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

#[cfg(feature = "permission-item")]
mod item {
    use super::*;
    use derivative::Derivative;
    use enum_map::Enum;
    #[cfg(feature = "database")]
    use sea_orm::FromQueryResult;
    use serde::{Deserialize, Serialize};
    use serde_repr::{Deserialize_repr, Serialize_repr};
    use std::{collections::HashMap, fmt};

    #[derive(Debug, Enum, Serialize_repr, Deserialize_repr)]
    #[repr(u8)]
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
    #[derive(Derivative, Clone, Serialize, Deserialize)]
    #[derivative(Default(new = "true"), Debug)]
    pub struct PermEntry {
        pub pid: HyUuid,
        pub perm: UserPerm,
    }

    impl PermEntry {
        pub(crate) fn script_check(&mut self, p: HashMap<HyUuid, PermissionItem>) -> bool {
            self.check(&p)
        }

        pub fn check(&self, p: &HashMap<HyUuid, PermissionItem>) -> bool {
            // root
            if p.contains_key(&ROOT_ID) {
                return true;
            }
            p.get(&self.pid)
                .is_some_and(|x| (x.perm & self.perm) == self.perm)
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
    #[derive(Debug, Derivative, Serialize, Deserialize, Clone)]
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
}

#[cfg(feature = "permission-checker")]
mod checker {
    use super::*;
    use derivative::Derivative;
    use parking_lot::RwLock;
    use rhai::{AST, Engine, Scope};
    use serde::{Deserialize, Serialize};
    use std::collections::HashMap;

    pub struct CheckerScript {
        engine: Engine,
        ast: AST,
    }

    impl CheckerScript {
        pub fn new(code: &str) -> Self {
            let mut engine = Engine::new();
            engine.register_fn("new_entry", Self::new_entry);
            engine.register_fn("check", PermEntry::script_check);
            let ast = engine.compile(code).unwrap();
            Self { engine, ast }
        }

        fn new_entry(s: &str, p: UserPerm) -> PermEntry {
            PermEntry {
                pid: HyUuid::parse(s).unwrap(),
                perm: p,
            }
        }

        pub fn check(&self, r: &HashMap<HyUuid, PermissionItem>) -> bool {
            let mut scope = Scope::new();
            scope.push_constant("PERM_REVOKE", PERM_REVOKE);
            scope.push_constant("PERM_FORBIDDEN", PERM_FORBIDDEN);
            scope.push_constant("PERM_NONE", PERM_NONE);
            scope.push_constant("PERM_WRITE", PERM_WRITE);
            scope.push_constant("PERM_READ", PERM_READ);
            scope.push_constant("PERM_ALL", PERM_ALL);
            scope.push_constant("PERMISSION", r.to_owned());
            self.engine
                .eval_ast_with_scope::<bool>(&mut scope, &self.ast)
                .unwrap()
        }
    }

    pub struct ScriptBuilder {
        code: String,
    }

    impl ScriptBuilder {
        pub fn new(id: HyUuid, perm: UserPerm) -> Self {
            Self {
                code: format!(r#"new_entry("{id}", {perm}).check(PERMISSION)"#),
            }
        }

        pub fn or_script(self, other: Self) -> Self {
            Self {
                code: format!(r#"({}) || ({})"#, self.code, other.code),
            }
        }

        pub fn or(self, id: HyUuid, perm: UserPerm) -> Self {
            self.or_script(Self::new(id, perm))
        }

        pub fn and_script(self, other: Self) -> Self {
            Self {
                code: format!(r#"({}) && ({})"#, self.code, other.code),
            }
        }

        pub fn and(self, id: HyUuid, perm: UserPerm) -> Self {
            self.and_script(Self::new(id, perm))
        }

        pub fn build(self) -> String {
            self.code
        }
    }

    #[derive(Serialize, Deserialize, Derivative)]
    #[derivative(Debug)]
    pub enum PermChecker {
        Entry(PermEntry),
        Script {
            code: String,
            #[serde(skip)]
            #[derivative(Debug = "ignore")]
            cache: RwLock<Option<CheckerScript>>,
        },
    }

    impl Default for PermChecker {
        fn default() -> Self {
            Self::Entry(PermEntry::new_user())
        }
    }

    impl Clone for PermChecker {
        fn clone(&self) -> Self {
            match self {
                Self::Entry(arg0) => Self::Entry(arg0.clone()),
                Self::Script { code, .. } => Self::Script {
                    code: code.clone(),
                    cache: Default::default(),
                },
            }
        }
    }

    impl PermChecker {
        pub fn new_entry(id: HyUuid, perm: UserPerm) -> Self {
            Self::Entry(PermEntry { pid: id, perm })
        }

        pub fn new_script(code: &str) -> Self {
            Self::Script {
                code: code.to_owned(),
                cache: Default::default(),
            }
        }

        pub fn check(&self, r: &HashMap<HyUuid, PermissionItem>) -> bool {
            match &self {
                PermChecker::Entry(x) => x.check(r),
                PermChecker::Script { code, cache } => {
                    let s = cache.read().is_none();
                    if s {
                        *cache.write() = Some(CheckerScript::new(code))
                    }
                    cache.read().as_ref().unwrap().check(r)
                }
            }
        }
    }
}
#[cfg(feature = "permission-checker")]
pub use checker::*;
#[cfg(feature = "permission-item")]
pub use item::*;
