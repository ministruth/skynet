use derivative::Derivative;
use skynet_api::{
    async_trait,
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
