use actix_cloud::macros::{entity_behavior, entity_id, entity_timestamp};
use sea_orm::entity::prelude::*;
use serde::Serialize;

use crate::HyUuid;

#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Eq, Default, Serialize)]
#[sea_orm(table_name = "groups")]
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

impl Related<super::users::Entity> for Entity {
    fn to() -> RelationDef {
        super::user_group_links::Relation::User.def()
    }

    fn via() -> Option<RelationDef> {
        Some(super::user_group_links::Relation::Group.def().rev())
    }
}

impl Related<super::permissions::Entity> for Entity {
    fn to() -> RelationDef {
        super::permission_links::Relation::Permission.def()
    }

    fn via() -> Option<RelationDef> {
        Some(super::permission_links::Relation::Group.def().rev())
    }
}

#[entity_id(HyUuid::new())]
#[entity_timestamp]
impl ActiveModel {}

#[entity_behavior]
impl ActiveModelBehavior for ActiveModel {}
