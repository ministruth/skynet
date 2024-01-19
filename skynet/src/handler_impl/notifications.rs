use std::sync::atomic::Ordering;

use anyhow::Result;
use async_trait::async_trait;
use derivative::Derivative;
use sea_orm::{ColumnTrait, EntityTrait, PaginatorTrait, QueryFilter};
use skynet::{entity::notifications, handler::NotificationHandler, UNREAD_NOTIFICATIONS};
use skynet_macro::default_handler_impl;

#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct DefaultNotificationHandler;

#[default_handler_impl(notifications)]
#[async_trait]
impl NotificationHandler for DefaultNotificationHandler {
    fn set_unread(&self, num: u64) {
        UNREAD_NOTIFICATIONS.store(num, Ordering::SeqCst);
    }

    fn add_unread(&self, num: u64) {
        UNREAD_NOTIFICATIONS.fetch_add(num, Ordering::SeqCst);
    }

    fn get_unread(&self) -> u64 {
        UNREAD_NOTIFICATIONS.load(Ordering::Relaxed)
    }
}
