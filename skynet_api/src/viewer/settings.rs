use std::collections::HashMap;

use crate::{entity::settings, request::Condition, HyUuid};
use anyhow::Result;
use base64::{prelude::BASE64_STANDARD, Engine};
use sea_orm::{
    ActiveModelTrait, ColumnTrait, ConnectionTrait, DatabaseTransaction, EntityTrait,
    IntoActiveModel, PaginatorTrait, QueryFilter, Set,
};
use skynet_macro::default_viewer;

pub struct SettingViewer;

#[default_viewer(settings)]
impl SettingViewer {
    pub async fn get_all<C>(db: &C) -> Result<HashMap<String, String>>
    where
        C: ConnectionTrait,
    {
        Ok(settings::Entity::find()
            .all(db)
            .await?
            .into_iter()
            .map(|x| (x.name, x.value))
            .collect())
    }

    pub async fn get<C>(db: &C, name: &str) -> Result<Option<String>>
    where
        C: ConnectionTrait,
    {
        Self::find_by_name(db, name)
            .await
            .map(|x| x.map(|x| x.value))
    }

    pub async fn get_or_init(db: &DatabaseTransaction, name: &str, value: &str) -> Result<String> {
        Self::find_or_init(db, name, value).await.map(|x| x.value)
    }

    pub async fn find_or_init(
        db: &DatabaseTransaction,
        name: &str,
        value: &str,
    ) -> Result<settings::Model> {
        match Self::find_by_name(db, name).await? {
            Some(x) => Ok(x),
            None => Ok(settings::ActiveModel {
                name: Set(name.to_owned()),
                value: Set(value.to_owned()),
                ..Default::default()
            }
            .insert(db)
            .await?),
        }
    }

    pub async fn find_by_name<C>(db: &C, name: &str) -> Result<Option<settings::Model>>
    where
        C: ConnectionTrait,
    {
        settings::Entity::find()
            .filter(settings::Column::Name.eq(name))
            .one(db)
            .await
            .map_err(Into::into)
    }

    pub async fn set(db: &DatabaseTransaction, name: &str, value: &str) -> Result<()> {
        if let Some(x) = Self::find_by_name(db, name).await? {
            if x.value != value {
                let mut x = x.into_active_model();
                x.value = Set(value.to_owned());
                x.save(db).await?;
            }
        } else {
            settings::ActiveModel {
                name: Set(name.to_owned()),
                value: Set(value.to_owned()),
                ..Default::default()
            }
            .insert(db)
            .await?;
        }
        Ok(())
    }

    pub async fn get_base64<C>(db: &C, name: &str) -> Result<Option<Vec<u8>>>
    where
        C: ConnectionTrait,
    {
        let v = Self::get(db, name).await?;
        if let Some(x) = v {
            Ok(Some(BASE64_STANDARD.decode(x)?))
        } else {
            Ok(None)
        }
    }

    pub async fn set_base64(db: &DatabaseTransaction, name: &str, value: &[u8]) -> Result<()> {
        Self::set(db, name, &BASE64_STANDARD.encode(value)).await
    }

    pub async fn delete<C>(db: &C, name: &str) -> Result<bool>
    where
        C: ConnectionTrait,
    {
        let rows = settings::Entity::delete_many()
            .filter(settings::Column::Name.eq(name))
            .exec(db)
            .await
            .map(|x| x.rows_affected)?;
        Ok(rows == 1)
    }
}
