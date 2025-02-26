use crate::{HyUuid, entity::webpush_subscriptions, hyuuid::uuids2strings, request::Condition};
use anyhow::Result;
use sea_orm::{
    ActiveModelTrait, ColumnTrait, ConnectionTrait, EntityTrait, PaginatorTrait, QueryFilter, Set,
};
use skynet_macro::default_viewer;

pub struct WebpushSubscriptionViewer;

#[default_viewer(webpush_subscriptions)]
impl WebpushSubscriptionViewer {
    pub async fn subscribe<C>(
        db: &C,
        uid: &HyUuid,
        topic: &HyUuid,
    ) -> Result<webpush_subscriptions::Model>
    where
        C: ConnectionTrait,
    {
        webpush_subscriptions::ActiveModel {
            uid: Set(*uid),
            topic: Set(*topic),
            ..Default::default()
        }
        .insert(db)
        .await
        .map_err(Into::into)
    }

    pub async fn unsubscribe<C>(db: &C, uid: &HyUuid, topic: &HyUuid) -> Result<u64>
    where
        C: ConnectionTrait,
    {
        webpush_subscriptions::Entity::delete_many()
            .filter(webpush_subscriptions::Column::Uid.eq(*uid))
            .filter(webpush_subscriptions::Column::Topic.eq(*topic))
            .exec(db)
            .await
            .map(|x| x.rows_affected)
            .map_err(Into::into)
    }
}
