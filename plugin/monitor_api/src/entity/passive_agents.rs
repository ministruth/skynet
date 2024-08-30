use serde::Serialize;
use skynet_api::{
    actix_cloud::{
        self,
        macros::{entity_behavior, entity_id, entity_timestamp},
    },
    sea_orm::{self, prelude::*},
};

use crate::HyUuid;

#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Eq, Default, Serialize)]
#[sea_orm(table_name = "2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa_passive_agents")]
pub struct Model {
    #[sea_orm(primary_key, auto_increment = false)]
    pub id: HyUuid,
    pub name: String,
    pub address: String,
    pub retry_time: i32,
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
