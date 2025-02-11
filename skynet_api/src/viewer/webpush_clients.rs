use crate::{entity::webpush_clients, hyuuid::uuids2strings, request::Condition, HyUuid};
use anyhow::Result;
use sea_orm::{
    ActiveModelTrait, ColumnTrait, ConnectionTrait, EntityTrait, PaginatorTrait, QueryFilter, Set,
};
use skynet_macro::default_viewer;

pub struct WebpushClientViewer;

#[default_viewer(webpush_clients)]
impl WebpushClientViewer {
    pub async fn create<C>(
        db: &C,
        uid: &HyUuid,
        endpoint: &str,
        p256dh: &str,
        auth: &str,
        lang: &str,
    ) -> Result<webpush_clients::Model>
    where
        C: ConnectionTrait,
    {
        webpush_clients::ActiveModel {
            uid: Set(*uid),
            endpoint: Set(endpoint.to_owned()),
            p256dh: Set(p256dh.to_owned()),
            auth: Set(auth.to_owned()),
            lang: Set(lang.to_owned()),
            ..Default::default()
        }
        .insert(db)
        .await
        .map_err(Into::into)
    }

    pub async fn find_by_endpoint<C>(
        db: &C,
        uid: &HyUuid,
        endpoint: &str,
    ) -> Result<Option<webpush_clients::Model>>
    where
        C: ConnectionTrait,
    {
        webpush_clients::Entity::find()
            .filter(webpush_clients::Column::Uid.eq(*uid))
            .filter(webpush_clients::Column::Endpoint.eq(endpoint))
            .one(db)
            .await
            .map_err(Into::into)
    }

    pub async fn delete_by_endpoint<C>(db: &C, uid: &HyUuid, endpoint: &str) -> Result<u64>
    where
        C: ConnectionTrait,
    {
        webpush_clients::Entity::delete_many()
            .filter(webpush_clients::Column::Uid.eq(*uid))
            .filter(webpush_clients::Column::Endpoint.eq(endpoint))
            .exec(db)
            .await
            .map(|x| x.rows_affected)
            .map_err(Into::into)
    }
}
