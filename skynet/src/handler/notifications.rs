use actix_cloud::async_trait;
use derivative::Derivative;
use skynet_api::{
    entity::notifications,
    handler::NotificationHandler,
    sea_orm::{ColumnTrait, EntityTrait, PaginatorTrait, QueryFilter},
};
use skynet_macro::default_handler_impl;

#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct DefaultNotificationHandler;

#[default_handler_impl(notifications)]
#[async_trait]
impl NotificationHandler for DefaultNotificationHandler {}
