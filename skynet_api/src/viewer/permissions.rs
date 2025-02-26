use crate::{
    HyUuid,
    entity::{groups, permission_links, permissions, users},
    hyuuid::uuids2strings,
    permission::{PERM_REVOKE, PermEntry, PermissionItem, UserPerm},
    request::Condition,
};
use anyhow::Result;
use sea_orm::{
    ActiveModelBehavior, ActiveModelTrait, ColumnTrait, ConnectionTrait, DatabaseTransaction,
    EntityTrait, JoinType::InnerJoin, PaginatorTrait, QueryFilter, QuerySelect, RelationTrait, Set,
};
use skynet_macro::default_viewer;

pub struct PermissionViewer;

#[default_viewer(permissions)]
impl PermissionViewer {
    pub async fn find_or_init(
        db: &DatabaseTransaction,
        name: &str,
        note: &str,
    ) -> Result<permissions::Model> {
        match permissions::Entity::find()
            .filter(permissions::Column::Name.eq(name))
            .one(db)
            .await?
        {
            Some(x) => Ok(x),
            None => Ok(permissions::ActiveModel {
                name: Set(name.to_owned()),
                note: Set(note.to_owned()),
                ..Default::default()
            }
            .insert(db)
            .await?),
        }
    }

    /// Find user `uid` permission.
    /// Note that user group permission is not included.
    pub async fn find_user<C>(db: &C, uid: &HyUuid) -> Result<Vec<PermissionItem>>
    where
        C: ConnectionTrait,
    {
        let ret = permission_links::Entity::find()
            .join(InnerJoin, permission_links::Relation::Permission.def())
            .join(InnerJoin, permission_links::Relation::User.def())
            .filter(permission_links::Column::Uid.eq(*uid))
            .column(permissions::Column::Name)
            .column(permissions::Column::Note)
            .column_as(users::Column::Username, "ug_name")
            .into_model::<PermissionItem>()
            .all(db)
            .await?;
        Ok(ret)
    }

    /// Find group `gid` permission.
    pub async fn find_group<C>(db: &C, gid: &HyUuid) -> Result<Vec<PermissionItem>>
    where
        C: ConnectionTrait,
    {
        let ret = permission_links::Entity::find()
            .join(InnerJoin, permission_links::Relation::Permission.def())
            .join(InnerJoin, permission_links::Relation::Group.def())
            .filter(permission_links::Column::Gid.eq(*gid))
            .column(permissions::Column::Name)
            .column(permissions::Column::Note)
            .column_as(groups::Column::Name, "ug_name")
            .into_model::<PermissionItem>()
            .all(db)
            .await?;
        Ok(ret)
    }

    /// Grant `uid` or `gid` with `pid` and `perm`.
    ///
    /// - If `uid` and `gid` both `None`, return `Ok` directly.
    /// - If `uid` and `gid` both `Some`, `pid` will be granted to all of them.
    pub async fn grant(
        db: &DatabaseTransaction,
        uid: Option<&HyUuid>,
        gid: Option<&HyUuid>,
        pid: &HyUuid,
        perm: UserPerm,
    ) -> Result<()> {
        if let Some(uid) = uid {
            Self::delete_user(db, uid, Some(pid)).await?;
            if perm != PERM_REVOKE {
                Self::create_user(db, uid, &[PermEntry { pid: *pid, perm }]).await?;
            }
        }
        if let Some(gid) = gid {
            Self::delete_group(db, gid, Some(pid)).await?;
            if perm != PERM_REVOKE {
                Self::create_group(db, gid, &[PermEntry { pid: *pid, perm }]).await?;
            }
        }
        Ok(())
    }

    /// Create new user `uid` permission `perm`.
    pub async fn create_user<C>(db: &C, uid: &HyUuid, perm: &[PermEntry]) -> Result<()>
    where
        C: ConnectionTrait,
    {
        if perm.is_empty() {
            return Ok(());
        }
        let mut ins = Vec::new();
        for i in perm {
            ins.push(
                permission_links::ActiveModel {
                    uid: Set(Some(*uid)),
                    pid: Set(i.pid),
                    perm: Set(i.perm),
                    ..Default::default()
                }
                .before_save(db, true)
                .await?,
            );
        }
        permission_links::Entity::insert_many(ins).exec(db).await?;
        Ok(())
    }

    /// Create new group `gid` permission `perm`.
    pub async fn create_group<C>(db: &C, gid: &HyUuid, perm: &[PermEntry]) -> Result<()>
    where
        C: ConnectionTrait,
    {
        if perm.is_empty() {
            return Ok(());
        }
        let mut ins = Vec::new();
        for i in perm {
            ins.push(
                permission_links::ActiveModel {
                    gid: Set(Some(*gid)),
                    pid: Set(i.pid),
                    perm: Set(i.perm),
                    ..Default::default()
                }
                .before_save(db, true)
                .await?,
            );
        }
        permission_links::Entity::insert_many(ins).exec(db).await?;
        Ok(())
    }

    /// Delete user `uid` permission `pid`.
    ///
    /// If `pid` is `None`, all user permission will be deleted.
    pub async fn delete_user<C>(db: &C, uid: &HyUuid, pid: Option<&HyUuid>) -> Result<u64>
    where
        C: ConnectionTrait,
    {
        let mut q =
            permission_links::Entity::delete_many().filter(permission_links::Column::Uid.eq(*uid));
        if let Some(x) = pid {
            q = q.filter(permission_links::Column::Pid.eq(*x));
        }
        Ok(q.exec(db).await?.rows_affected)
    }

    /// Delete group `gid` permission `pid`.
    ///
    /// If `pid` is `None`, all group permission will be deleted.
    pub async fn delete_group<C>(db: &C, gid: &HyUuid, pid: Option<&HyUuid>) -> Result<u64>
    where
        C: ConnectionTrait,
    {
        let mut q =
            permission_links::Entity::delete_many().filter(permission_links::Column::Gid.eq(*gid));
        if let Some(x) = pid {
            q = q.filter(permission_links::Column::Pid.eq(*x));
        }
        Ok(q.exec(db).await?.rows_affected)
    }
}
