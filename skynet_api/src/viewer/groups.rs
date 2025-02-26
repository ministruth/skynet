use crate::{
    HyUuid,
    entity::{groups, user_group_links, users},
    hyuuid::uuids2strings,
    request::{Condition, SelectPage},
};
use anyhow::Result;
use sea_orm::{
    ActiveModelBehavior, ActiveModelTrait, ActiveValue::NotSet, ColumnTrait, ConnectionTrait,
    EntityTrait, JoinType::InnerJoin, ModelTrait, PaginatorTrait, QueryFilter, QuerySelect,
    RelationTrait, Set, Unchanged,
};
use skynet_macro::default_viewer;

pub struct GroupViewer;

#[default_viewer(groups)]
impl GroupViewer {
    /// Find group by `name`.
    pub async fn find_by_name<C>(db: &C, name: &str) -> Result<Option<groups::Model>>
    where
        C: ConnectionTrait,
    {
        groups::Entity::find()
            .filter(groups::Column::Name.eq(name))
            .one(db)
            .await
            .map_err(Into::into)
    }

    /// Find group `gid` user by `uid`.
    pub async fn find_group_user_by_id<C>(
        db: &C,
        gid: &HyUuid,
        uid: &HyUuid,
    ) -> Result<Option<users::Model>>
    where
        C: ConnectionTrait,
    {
        users::Entity::find()
            .filter(user_group_links::Column::Uid.eq(*uid))
            .filter(user_group_links::Column::Gid.eq(*gid))
            .join(InnerJoin, user_group_links::Relation::User.def().rev())
            .one(db)
            .await
            .map_err(Into::into)
    }

    /// Find group `gid` user.
    pub async fn find_group_user<C>(
        db: &C,
        gid: &HyUuid,
        cond: Condition,
    ) -> Result<(Vec<users::Model>, u64)>
    where
        C: ConnectionTrait,
    {
        let (q, page) = cond
            .add(user_group_links::Column::Gid.eq(*gid))
            .build(users::Entity::find());
        q.join(InnerJoin, user_group_links::Relation::User.def().rev())
            .select_page(db, page)
            .await
    }

    /// Find user `uid` group.
    pub async fn find_user_group<C>(db: &C, uid: &HyUuid, join: bool) -> Result<Vec<groups::Model>>
    where
        C: ConnectionTrait,
    {
        if join {
            users::Model {
                id: *uid,
                ..Default::default()
            }
            .find_linked(user_group_links::UserToGroup)
            .all(db)
            .await
            .map_err(Into::into)
        } else {
            let mut ret = Vec::new();
            let group = user_group_links::Entity::find()
                .filter(user_group_links::Column::Uid.eq(*uid))
                .all(db)
                .await?;
            for i in group {
                ret.push(groups::Model {
                    id: i.gid,
                    ..Default::default()
                });
            }
            Ok(ret)
        }
    }

    /// Link all `uid` user to all `gid` group.
    pub async fn link<C>(db: &C, uid: &[HyUuid], gid: &[HyUuid]) -> Result<()>
    where
        C: ConnectionTrait,
    {
        if uid.is_empty() || gid.is_empty() {
            return Ok(());
        }
        let mut ins = Vec::new();
        for i in uid {
            for j in gid {
                ins.push(
                    user_group_links::ActiveModel {
                        uid: Set(i.to_owned()),
                        gid: Set(j.to_owned()),
                        ..Default::default()
                    }
                    .before_save(db, true) // not invoke for batch insert
                    .await?,
                );
            }
        }
        user_group_links::Entity::insert_many(ins).exec(db).await?;
        Ok(())
    }

    /// Unlinks user and group.
    ///
    /// - If `uid.is_empty()`, remove all users in each `gid`.
    /// - If `gid.is_empty()`, remove all groups in each `uid`.
    /// - Otherwise remove each `uid` with each `gid`.
    pub async fn unlink<C>(db: &C, uid: &[HyUuid], gid: &[HyUuid]) -> Result<u64>
    where
        C: ConnectionTrait,
    {
        if uid.is_empty() && gid.is_empty() {
            Ok(0)
        } else {
            let res = if uid.is_empty() {
                user_group_links::Entity::delete_many()
                    .filter(user_group_links::Column::Gid.is_in(uuids2strings(gid)))
                    .exec(db)
                    .await?
            } else if gid.is_empty() {
                user_group_links::Entity::delete_many()
                    .filter(user_group_links::Column::Uid.is_in(uuids2strings(uid)))
                    .exec(db)
                    .await?
            } else {
                user_group_links::Entity::delete_many()
                    .filter(user_group_links::Column::Uid.is_in(uuids2strings(uid)))
                    .filter(user_group_links::Column::Gid.is_in(uuids2strings(gid)))
                    .exec(db)
                    .await?
            };
            Ok(res.rows_affected)
        }
    }

    /// Update group infos by `gid`.
    pub async fn update<C>(
        db: &C,
        gid: &HyUuid,
        name: Option<&str>,
        note: Option<&str>,
    ) -> Result<groups::Model>
    where
        C: ConnectionTrait,
    {
        groups::ActiveModel {
            id: Unchanged(*gid),
            name: name.map_or(NotSet, |x| Set(x.to_owned())),
            note: note.map_or(NotSet, |x| Set(x.to_owned())),
            ..Default::default()
        }
        .update(db)
        .await
        .map_err(Into::into)
    }

    /// Create new user group.
    pub async fn create<C>(db: &C, name: &str, note: &str) -> Result<groups::Model>
    where
        C: ConnectionTrait,
    {
        groups::ActiveModel {
            name: Set(name.to_owned()),
            note: Set(note.to_owned()),
            ..Default::default()
        }
        .insert(db)
        .await
        .map_err(Into::into)
    }
}
