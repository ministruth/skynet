use serde::Serialize;
use skynet::sea_orm::{self, prelude::*};
use skynet_macro::{entity_behavior, entity_id, entity_timestamp};

use crate::HyUuid;

#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Eq, Default, Serialize)]
#[sea_orm(table_name = "agents")]
pub struct Model {
    #[sea_orm(primary_key, auto_increment = false)]
    pub id: HyUuid,
    pub uid: String,
    pub name: String,
    pub os: Option<String>,
    pub hostname: Option<String>,
    pub ip: String,
    pub system: Option<String>,
    pub machine: Option<String>,
    pub last_login: i64,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Copy, Clone, Debug, EnumIter, DeriveRelation)]
pub enum Relation {
    #[sea_orm(has_many = "super::agent_settings::Entity")]
    Setting,
}

impl Related<super::agent_settings::Entity> for Entity {
    fn to() -> RelationDef {
        Relation::Setting.def()
    }
}

#[entity_id]
#[entity_timestamp]
impl ActiveModel {}

#[entity_behavior]
impl ActiveModelBehavior for ActiveModel {}
