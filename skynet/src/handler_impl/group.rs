use anyhow::Result;
use async_trait::async_trait;
use derivative::Derivative;
use sea_orm::{
    ActiveModelBehavior, ActiveModelTrait, ActiveValue::NotSet, ColumnTrait, DatabaseTransaction,
    EntityTrait, JoinType::InnerJoin, ModelTrait, PaginatorTrait, QueryFilter, QuerySelect,
    RelationTrait, Set, Unchanged,
};
use skynet::{
    entity::{groups, user_group_links, users},
    handler::GroupHandler,
    hyuuid::uuid2string,
    Condition, HyUuid,
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
            .map_err(anyhow::Error::from)
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
        let q = q.join(InnerJoin, user_group_links::Relation::User.def().rev());
        if let Some(page) = page {
            let q = q.paginate(db, page.size);
            Ok((q.fetch_page(page.page - 1).await?, q.num_items().await?))
        } else {
            let res = q.all(db).await?;
            let cnt = res.len() as u64;
            Ok((res, cnt))
        }
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
            .map_err(anyhow::Error::from)
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
            .map_err(anyhow::Error::from)
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
        .map_err(anyhow::Error::from)
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
        .map_err(anyhow::Error::from)
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
                    .filter(user_group_links::Column::Gid.is_in(uuid2string(gid)))
                    .exec(db)
                    .await?
            } else if gid.is_empty() {
                user_group_links::Entity::delete_many()
                    .filter(user_group_links::Column::Uid.is_in(uuid2string(uid)))
                    .exec(db)
                    .await?
            } else {
                user_group_links::Entity::delete_many()
                    .filter(user_group_links::Column::Uid.is_in(uuid2string(uid)))
                    .filter(user_group_links::Column::Gid.is_in(uuid2string(gid)))
                    .exec(db)
                    .await?
            };
            Ok(res.rows_affected)
        }
    }
}
