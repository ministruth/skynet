use actix_cloud::{async_trait, Result};
use derivative::Derivative;
use skynet_api::{
    entity::{groups, user_group_links, users},
    handler::GroupHandler,
    hyuuid::uuids2strings,
    request::{Condition, SelectPage},
    sea_orm::{
        ActiveModelBehavior, ActiveModelTrait, ActiveValue::NotSet, ColumnTrait,
        DatabaseTransaction, EntityTrait, JoinType::InnerJoin, ModelTrait, PaginatorTrait,
        QueryFilter, QuerySelect, RelationTrait, Set, Unchanged,
    },
    HyUuid,
};
use skynet_macro::default_handler_impl;

#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct DefaultGroupHandler;

#[default_handler_impl(groups)]
#[async_trait]
impl GroupHandler for DefaultGroupHandler {
    async fn find_user_group(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
        join: bool,
    ) -> Result<Vec<groups::Model>> {
        if join {
            users::Model {
                id: uid.to_owned(),
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

    async fn find_group_user(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        cond: Condition,
    ) -> Result<(Vec<users::Model>, u64)> {
        let (q, page) = cond
            .add(user_group_links::Column::Gid.eq(*gid))
            .build(users::Entity::find());
        q.join(InnerJoin, user_group_links::Relation::User.def().rev())
            .select_page(db, page)
            .await
    }

    async fn find_group_user_by_id(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        uid: &HyUuid,
    ) -> Result<Option<users::Model>> {
        users::Entity::find()
            .filter(user_group_links::Column::Uid.eq(*uid))
            .filter(user_group_links::Column::Gid.eq(*gid))
            .join(InnerJoin, user_group_links::Relation::User.def().rev())
            .one(db)
            .await
            .map_err(Into::into)
    }

    async fn find_by_name(
        &self,
        db: &DatabaseTransaction,
        name: &str,
    ) -> Result<Option<groups::Model>> {
        groups::Entity::find()
            .filter(groups::Column::Name.eq(name))
            .one(db)
            .await
            .map_err(Into::into)
    }

    async fn update(
        &self,
        db: &DatabaseTransaction,
        gid: &HyUuid,
        name: Option<&str>,
        note: Option<&str>,
    ) -> Result<groups::Model> {
        groups::ActiveModel {
            id: Unchanged(gid.to_owned()),
            name: name.map_or(NotSet, |x| Set(x.to_owned())),
            note: note.map_or(NotSet, |x| Set(x.to_owned())),
            ..Default::default()
        }
        .update(db)
        .await
        .map_err(Into::into)
    }

    async fn create(
        &self,
        db: &DatabaseTransaction,
        name: &str,
        note: &str,
    ) -> Result<groups::Model> {
        groups::ActiveModel {
            name: Set(name.to_owned()),
            note: Set(note.to_owned()),
            ..Default::default()
        }
        .insert(db)
        .await
        .map_err(Into::into)
    }

    async fn link(&self, db: &DatabaseTransaction, uid: &[HyUuid], gid: &[HyUuid]) -> Result<()> {
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

    async fn unlink(
        &self,
        db: &DatabaseTransaction,
        uid: &[HyUuid],
        gid: &[HyUuid],
    ) -> Result<u64> {
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
}
