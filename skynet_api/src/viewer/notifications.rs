use crate::{entity::notifications, hyuuid::uuids2strings, request::Condition, HyUuid};
use sea_orm::{ColumnTrait, EntityTrait, PaginatorTrait, QueryFilter};
use skynet_macro::default_viewer;

pub struct NotificationViewer;

#[default_viewer(notifications)]
impl NotificationViewer {}
