use anyhow::Result;
use async_trait::async_trait;
use derivative::Derivative;
use sea_orm::{ColumnTrait, EntityTrait, PaginatorTrait, QueryFilter};
use skynet::{entity::notifications, handler::NotificationHandler};
use skynet_macro::default_handler_impl;

#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct DefaultNotificationHandler;

#[default_handler_impl(notifications)]
#[async_trait]
impl NotificationHandler for DefaultNotificationHandler {}
