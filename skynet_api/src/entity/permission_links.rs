use actix_cloud::macros::{entity_id, entity_timestamp};
use sea_orm::entity::prelude::*;
use serde::Serialize;

use crate::HyUuid;

#[derive(Clone, Debug, PartialEq, DeriveEntityModel, Eq, Default, Serialize)]
#[sea_orm(table_name = "permission_links")]
pub struct Model {
    #[sea_orm(primary_key, auto_increment = false)]
    pub id: HyUuid,
    pub uid: Option<HyUuid>,
    pub gid: Option<HyUuid>,
    pub pid: HyUuid,
    pub perm: i32,
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
    #[sea_orm(
        belongs_to = "super::permissions::Entity",
        from = "Column::Pid",
        to = "super::permissions::Column::Id"
    )]
    Permission,
}

#[entity_id(HyUuid::new())]
#[entity_timestamp]
impl ActiveModel {}

#[async_trait::async_trait]
impl ActiveModelBehavior for ActiveModel {
    async fn before_save<C>(self, _: &C, insert: bool) -> Result<Self, DbErr>
    where
        C: ConnectionTrait,
    {
        if self.uid.is_not_set() && self.gid.is_not_set() {
            return Err(DbErr::Custom(
                "uid and gid should not be both NULL".to_owned(),
            ));
        }
        let mut new = self.clone();
        self.entity_id(&mut new, insert);
        self.entity_timestamp(&mut new, insert);
        Ok(new)
    }
}

#[derive(Debug)]
pub struct UserToPermission;

impl Linked for UserToPermission {
    type FromEntity = super::users::Entity;

    type ToEntity = super::permissions::Entity;

    fn link(&self) -> Vec<RelationDef> {
        vec![
            super::permission_links::Relation::User.def().rev(),
            super::permission_links::Relation::Permission.def(),
        ]
    }
}

#[derive(Debug)]
pub struct GroupToPermission;

impl Linked for GroupToPermission {
    type FromEntity = super::groups::Entity;

    type ToEntity = super::permissions::Entity;

    fn link(&self) -> Vec<RelationDef> {
        vec![
            super::permission_links::Relation::Group.def().rev(),
            super::permission_links::Relation::Permission.def(),
        ]
    }
}
