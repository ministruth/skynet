use actix_cloud_codegen::{entity_behavior, entity_id, entity_timestamp};
use sea_orm::entity::prelude::*;
use serde::{Deserialize, Serialize};

use crate::HyUuid;

#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Eq, Default, Serialize, Deserialize)]
#[sea_orm(table_name = "permissions")]
pub struct Model {
    #[sea_orm(primary_key, auto_increment = false)]
    pub id: HyUuid,
    pub name: String,
    pub note: String,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Copy, Clone, Debug, EnumIter, DeriveRelation)]
pub enum Relation {}

#[entity_id(HyUuid::new())]
#[entity_timestamp]
impl ActiveModel {}

#[entity_behavior]
impl ActiveModelBehavior for ActiveModel {}
