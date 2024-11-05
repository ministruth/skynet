use actix_cloud_codegen::{entity_behavior, entity_id, entity_timestamp};
use sea_orm::entity::prelude::*;
use serde::Serialize;

use crate::{serializer::vec_string_option, HyUuid};

#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Eq, Default, Serialize)]
#[sea_orm(table_name = "users")]
pub struct Model {
    #[sea_orm(primary_key, auto_increment = false)]
    pub id: HyUuid,
    pub username: String,
    #[serde(skip)]
    pub password: String,
    #[serde(serialize_with = "vec_string_option")]
    pub avatar: Option<Vec<u8>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_login: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_ip: Option<String>,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Copy, Clone, Debug, EnumIter, DeriveRelation)]
pub enum Relation {}

impl Related<super::groups::Entity> for Entity {
    fn to() -> RelationDef {
        super::user_group_links::Relation::Group.def()
    }

    fn via() -> Option<RelationDef> {
        Some(super::user_group_links::Relation::User.def().rev())
    }
}

impl Related<super::permissions::Entity> for Entity {
    fn to() -> RelationDef {
        super::permission_links::Relation::Permission.def()
    }

    fn via() -> Option<RelationDef> {
        Some(super::permission_links::Relation::User.def().rev())
    }
}

#[entity_id(HyUuid::new())]
#[entity_timestamp]
impl ActiveModel {}

#[entity_behavior]
impl ActiveModelBehavior for ActiveModel {}
