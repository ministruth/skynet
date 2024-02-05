use anyhow::Result;
use async_trait::async_trait;
use derivative::Derivative;
use migration::JoinType;
use sea_orm::{
    ActiveModelBehavior, ActiveModelTrait, ColumnTrait, DatabaseTransaction, EntityTrait,
    PaginatorTrait, QueryFilter, QuerySelect, RelationTrait, Set,
};
use skynet::{
    entity::{groups, permission_links, permissions, users},
    handler::PermHandler,
    permission::{PermEntry, PermissionItem, UserPerm, PERM_REVOKE},
    HyUuid,
};
use skynet_macro::default_handler_impl;

#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct DefaultPermHandler;

#[default_handler_impl(permissions)]
#[async_trait]
impl PermHandler for DefaultPermHandler {
    async fn find_or_init(
        &self,
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

    async fn find_user(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
    ) -> Result<Vec<PermissionItem>> {
        let ret = permission_links::Entity::find()
            .join(
                JoinType::InnerJoin,
                permission_links::Relation::Permission.def(),
            )
            .join(JoinType::InnerJoin, permission_links::Relation::User.def())
            .filter(permission_links::Column::Uid.eq(*uid))
            .column(permissions::Column::Name)
            .column(permissions::Column::Note)
            .column_as(users::Column::Username, "ug_name")
            .into_model::<PermissionItem>()
            .all(db)
            .await?;
        Ok(ret)
    }

    async fn find_group(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
    ) -> Result<Vec<PermissionItem>> {
        let ret = permission_links::Entity::find()
            .join(
                JoinType::InnerJoin,
                permission_links::Relation::Permission.def(),
            )
            .join(JoinType::InnerJoin, permission_links::Relation::Group.def())
            .filter(permission_links::Column::Gid.eq(*gid))
            .column(permissions::Column::Name)
            .column(permissions::Column::Note)
            .column_as(groups::Column::Name, "ug_name")
            .into_model::<PermissionItem>()
            .all(db)
            .await?;
        Ok(ret)
    }

    async fn grant(
        &self,
        db: &DatabaseTransaction,
        uid: Option<&HyUuid>,
        gid: Option<&HyUuid>,
        pid: &HyUuid,
        perm: UserPerm,
    ) -> Result<()> {
        if let Some(uid) = uid {
            self.delete_user(db, uid, Some(pid)).await?;
            if perm != PERM_REVOKE {
                self.create_user(db, uid, &[PermEntry { pid: *pid, perm }])
                    .await?;
            }
        }
        if let Some(gid) = gid {
            self.delete_group(db, gid, Some(pid)).await?;
            if perm != PERM_REVOKE {
                self.create_group(db, gid, &[PermEntry { pid: *pid, perm }])
                    .await?;
            }
        }
        Ok(())
    }

    async fn create_user(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
        perm: &[PermEntry],
    ) -> Result<()> {
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

    async fn create_group(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        perm: &[PermEntry],
    ) -> Result<()> {
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

    async fn delete_user(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
        pid: Option<&HyUuid>,
    ) -> Result<u64> {
        let mut q =
            permission_links::Entity::delete_many().filter(permission_links::Column::Uid.eq(*uid));
        if let Some(x) = pid {
            q = q.filter(permission_links::Column::Pid.eq(*x));
        }
        Ok(q.exec(db).await?.rows_affected)
    }

    async fn delete_group(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        pid: Option<&HyUuid>,
    ) -> Result<u64> {
        let mut q =
            permission_links::Entity::delete_many().filter(permission_links::Column::Gid.eq(*gid));
        if let Some(x) = pid {
            q = q.filter(permission_links::Column::Pid.eq(*x));
        }
        Ok(q.exec(db).await?.rows_affected)
    }
}
