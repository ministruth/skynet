//! `SeaORM` Entity. Generated by sea-orm-codegen 0.10.3

use sea_orm::entity::prelude::*;
use serde::Serialize;
use skynet_macro::{entity_behavior, entity_id, entity_timestamp};

use crate::HyUuid;

#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Eq, Default, Serialize)]
#[sea_orm(table_name = "settings")]
pub struct Model {
    #[sea_orm(primary_key, auto_increment = false)]
    pub id: HyUuid,
    pub name: String,
    pub value: String,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Copy, Clone, Debug, EnumIter, DeriveRelation)]
pub enum Relation {}

#[entity_id]
#[entity_timestamp]
impl ActiveModel {}

#[entity_behavior]
impl ActiveModelBehavior for ActiveModel {}