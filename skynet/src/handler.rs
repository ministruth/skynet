use crate::{
    entity::{groups, notifications, permissions, users},
    permission::UserPerm,
    Condition, HyUuid, PermEntry, PermissionItem, Skynet,
};
use anyhow::Result;
use async_trait::async_trait;
use redis::aio::ConnectionManager;
use sea_orm::DatabaseTransaction;
use skynet_macro::default_handler;
use std::collections::HashMap;

/// User handler.
#[default_handler(users)]
#[async_trait]
pub trait UserHandler: Send + Sync {
    /// Create new user, generate random password if not set.
    /// Returned password is in plain text.
    async fn create(
        &self,
        db: &DatabaseTransaction,
        username: &str,
        password: Option<&str>,
        avatar: Option<Vec<u8>>,
    ) -> Result<users::Model>;

    /// Update user infos by `uid`.
    /// Password will be automatically hashed.
    ///
    /// User will be kicked if password is updated.
    async fn update(
        &self,
        db: &DatabaseTransaction,
        redis: &ConnectionManager,
        skynet: &Skynet,
        uid: &HyUuid,
        username: Option<&str>,
        password: Option<&str>,
        avatar: Option<Vec<u8>>,
    ) -> Result<users::Model>;

    /// Update login `uid` and `ip`.
    async fn update_login(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
        ip: &str,
    ) -> Result<users::Model>;

    /// Check `username` `password`.
    ///
    /// - If error, return `Err`.
    /// - If `username` not found, return `(false, None)`.
    /// - If `password` not match, return `(false, Some)`.
    /// - If success, return `(true, Some)`.
    async fn check_pass(
        &self,
        db: &DatabaseTransaction,
        username: &str,
        password: &str,
    ) -> Result<(bool, Option<users::Model>)>;

    /// Find user by `username`.
    /// Return `None` when not found.
    async fn find_by_name(
        &self,
        db: &DatabaseTransaction,
        username: &str,
    ) -> Result<Option<users::Model>>;

    /// Reset user password by `uid`.
    /// Return `None` when not found, password is in plain text otherwise.
    ///
    /// User will be kicked after reset.
    async fn reset(
        &self,
        db: &DatabaseTransaction,
        redis: &ConnectionManager,
        skynet: &Skynet,
        uid: &HyUuid,
    ) -> Result<Option<users::Model>>;

    /// Kick all `uid` login sessions.
    async fn kick(&self, redis: &ConnectionManager, skynet: &Skynet, uid: &HyUuid) -> Result<()>;

    /// Delete all users and kick them.
    async fn delete_all(
        &self,
        db: &DatabaseTransaction,
        redis: &ConnectionManager,
        skynet: &Skynet,
    ) -> Result<u64>;

    /// Delete `uid` users and kick them.
    async fn delete(
        &self,
        db: &DatabaseTransaction,
        redis: &ConnectionManager,
        skynet: &Skynet,
        uid: &[HyUuid],
    ) -> Result<u64>;
}

/// Group handler.
#[default_handler(groups)]
#[async_trait]
pub trait GroupHandler: Send + Sync {
    /// Link all `uid` user to all `gid` group.
    async fn link(&self, db: &DatabaseTransaction, uid: &[HyUuid], gid: &[HyUuid]) -> Result<()>;

    /// Unlinks user and group.
    ///
    /// - If `uid.is_empty()`, remove all users in each `gid`.
    /// - If `gid.is_empty()`, remove all groups in each `uid`.
    /// - Otherwise remove each `uid` with each `gid`.
    async fn unlink(&self, db: &DatabaseTransaction, uid: &[HyUuid], gid: &[HyUuid])
        -> Result<u64>;

    /// Update group infos by `gid`.
    async fn update(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        name: Option<&str>,
        note: Option<&str>,
    ) -> Result<groups::Model>;

    /// Create new user group.
    async fn create(
        &self,
        db: &DatabaseTransaction,
        name: &str,
        note: &str,
    ) -> Result<groups::Model>;

    /// Find group by `name`.
    async fn find_by_name(
        &self,
        db: &DatabaseTransaction,
        name: &str,
    ) -> Result<Option<groups::Model>>;

    /// Find group `gid` user by `uid`.
    async fn find_group_user_by_id(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        uid: &HyUuid,
    ) -> Result<Option<users::Model>>;

    /// Find group `gid` user.
    async fn find_group_user(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        cond: Condition,
    ) -> Result<(Vec<users::Model>, u64)>;

    /// Find user `uid` group.
    async fn find_user_group(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
        join: bool,
    ) -> Result<Vec<groups::Model>>;
}

/// Permission handler.
#[default_handler(permissions)]
#[async_trait]
pub trait PermHandler: Send + Sync {
    async fn find_or_init(
        &self,
        db: &DatabaseTransaction,
        name: &str,
        note: &str,
    ) -> Result<permissions::Model>;

    /// Find user `uid` permission.
    /// Note that user group permission is not included.
    async fn find_user(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
    ) -> Result<Vec<PermissionItem>>;

    /// Find group `gid` permission.
    async fn find_group(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
    ) -> Result<Vec<PermissionItem>>;

    /// Grant `uid` or `gid` with `pid` and `perm`.
    ///
    /// - If `uid` and `gid` both `None`, return `Ok` directly.
    /// - If `uid` and `gid` both `Some`, `pid` will be granted to all of them.
    async fn grant(
        &self,
        db: &DatabaseTransaction,
        uid: Option<&HyUuid>,
        gid: Option<&HyUuid>,
        pid: &HyUuid,
        perm: UserPerm,
    ) -> Result<()>;

    /// Create new user `uid` permission `perm`.
    async fn create_user(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
        perm: &[PermEntry],
    ) -> Result<()>;

    /// Create new group `gid` permission `perm`.
    async fn create_group(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        perm: &[PermEntry],
    ) -> Result<()>;

    /// Delete user `uid` permission `pid`.
    ///
    /// If `pid` is `None`, all user permission will be deleted.
    async fn delete_user(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
        pid: Option<&HyUuid>,
    ) -> Result<u64>;

    /// Delete group `gid` permission `pid`.
    ///
    /// If `pid` is `None`, all group permission will be deleted.
    async fn delete_group(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        pid: Option<&HyUuid>,
    ) -> Result<u64>;
}

/// Notification handler.
#[default_handler(notifications)]
#[async_trait]
pub trait NotificationHandler: Send + Sync {}

#[async_trait]
pub trait SettingHandler: Send + Sync {
    /// Build setting cache.
    async fn build_cache(&self, db: &DatabaseTransaction) -> Result<()>;

    /// Get all settings.
    fn get_all(&self) -> HashMap<String, String>;

    /// Get `name` setting.
    fn get(&self, name: &str) -> Option<String>;

    /// Set setting `name` with `value`.
    async fn set(&self, db: &DatabaseTransaction, name: &str, value: &str) -> Result<()>;

    /// Delete `name` setting.
    async fn delete(&self, db: &DatabaseTransaction, name: &str) -> Result<bool>;

    /// Delete all settings.
    async fn delete_all(&self, db: &DatabaseTransaction) -> Result<u64>;
}
