use actix_cloud_codegen::{entity_behavior, entity_id, entity_timestamp};
use sea_orm::entity::prelude::*;
use serde::{Deserialize, Serialize};

use crate::HyUuid;

#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Eq, Default, Serialize, Deserialize)]
#[sea_orm(table_name = "user_group_links")]
pub struct Model {
    #[sea_orm(primary_key, auto_increment = false)]
    pub id: HyUuid,
    pub uid: HyUuid,
    pub gid: HyUuid,
    pub created_at: i64,
    pub updated_at: i64,
}

#[derive(Copy, Clone, Debug, EnumIter, DeriveRelation)]
pub enum Relation {
    #[sea_orm(
        belongs_to = "super::users::Entity",
        from = "Column::Uid",
        to = "super::users::Column::Id"
    )]
    User,
    #[sea_orm(
        belongs_to = "super::groups::Entity",
        from = "Column::Gid",
        to = "super::groups::Column::Id"
    )]
    Group,
}

#[entity_id(HyUuid::new())]
#[entity_timestamp]
impl ActiveModel {}

#[entity_behavior]
impl ActiveModelBehavior for ActiveModel {}

#[derive(Debug)]
pub struct UserToGroup;

impl Linked for UserToGroup {
    type FromEntity = super::users::Entity;

    type ToEntity = super::groups::Entity;

    fn link(&self) -> Vec<RelationDef> {
        vec![
            super::user_group_links::Relation::User.def().rev(),
            super::user_group_links::Relation::Group.def(),
        ]
    }
}

#[derive(Debug)]
pub struct GroupToUser;

impl Linked for GroupToUser {
    type FromEntity = super::groups::Entity;

    type ToEntity = super::users::Entity;

    fn link(&self) -> Vec<RelationDef> {
        vec![
            super::user_group_links::Relation::Group.def().rev(),
            super::user_group_links::Relation::User.def(),
        ]
    }
}
