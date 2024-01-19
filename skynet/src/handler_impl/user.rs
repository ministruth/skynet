use std::time;

use anyhow::Result;
use async_trait::async_trait;
use derivative::Derivative;
use redis::AsyncCommands;
use sea_orm::{
    ActiveModelTrait, ActiveValue::NotSet, ColumnTrait, DatabaseTransaction, EntityTrait,
    PaginatorTrait, QueryFilter, Set, Unchanged,
};
use sha3::{digest::Digest, Sha3_512};
use skynet::{entity::users, handler::UserHandler, hyuuid::uuid2string, utils, HyUuid, Skynet};
use skynet_macro::default_handler_impl;

#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct DefaultUserHandler;

impl DefaultUserHandler {
    fn hash_pass(pass: &str, prefix: &str, suffix: &str) -> String {
        format!("{:x}", Sha3_512::digest(format!("{prefix}{pass}{suffix}")))
    }
}

#[default_handler_impl(users)]
#[async_trait]
impl UserHandler for DefaultUserHandler {
    async fn create(
        &self,
        db: &DatabaseTransaction,
        skynet: &Skynet,
        username: &str,
        password: Option<&str>,
        avatar: Option<Vec<u8>>,
    ) -> Result<users::Model> {
        let password = password.map_or_else(|| utils::rand_string(16), ToOwned::to_owned);
        let prefix =
            utils::rand_string_all(skynet.config.database_salt_prefix.get().try_into().unwrap());
        let suffix =
            utils::rand_string_all(skynet.config.database_salt_suffix.get().try_into().unwrap());
        let mut user = users::ActiveModel {
            username: Set(username.to_owned()),
            password: Set(Self::hash_pass(&password, &prefix, &suffix)),
            salt_prefix: Set(prefix),
            salt_suffix: Set(suffix),
            avatar: Set(avatar),
            ..Default::default()
        };
        if username == "root" {
            user.id = Set(HyUuid::nil());
        }
        let mut user = user.insert(db).await?;
        user.password = password;
        Ok(user)
    }

    async fn update(
        &self,
        db: &DatabaseTransaction,
        skynet: &Skynet,
        uid: &HyUuid,
        username: Option<&str>,
        password: Option<&str>,
        avatar: Option<Vec<u8>>,
        salt_prefix: &str,
        salt_suffix: &str,
    ) -> Result<users::Model> {
        if password.is_some() {
            self.kick(skynet, uid).await?;
        }
        users::ActiveModel {
            id: Unchanged(uid.to_owned()),
            username: username.map_or(NotSet, |x| Set(x.to_owned())),
            password: password.map_or(NotSet, |x| {
                Set(Self::hash_pass(x, salt_prefix, salt_suffix))
            }),
            avatar: avatar.map_or(NotSet, |x| {
                if x.is_empty() {
                    Set(None)
                } else {
                    Set(Some(x))
                }
            }),
            ..Default::default()
        }
        .update(db)
        .await
        .map_err(anyhow::Error::from)
    }

    async fn update_login(
        &self,
        db: &DatabaseTransaction,
        uid: &HyUuid,
        ip: &str,
    ) -> Result<users::Model> {
        let now: i64 = time::SystemTime::now()
            .duration_since(time::UNIX_EPOCH)
            .unwrap()
            .as_millis()
            .try_into()
            .unwrap();
        users::ActiveModel {
            id: Unchanged(uid.to_owned()),
            last_ip: Set(Some(ip.to_owned())),
            last_login: Set(Some(now)),
            ..Default::default()
        }
        .update(db)
        .await
        .map_err(anyhow::Error::from)
    }

    async fn check_pass(
        &self,
        db: &DatabaseTransaction,
        username: &str,
        password: &str,
    ) -> Result<(bool, Option<users::Model>)> {
        let user = self.find_by_name(db, username).await?;
        match user {
            Some(user) => {
                if user.password == Self::hash_pass(password, &user.salt_prefix, &user.salt_suffix)
                {
                    Ok((true, Some(user)))
                } else {
                    Ok((false, Some(user)))
                }
            }
            None => Ok((false, None)),
        }
    }

    async fn find_by_name(
        &self,
        db: &DatabaseTransaction,
        username: &str,
    ) -> Result<Option<users::Model>> {
        users::Entity::find()
            .filter(users::Column::Username.eq(username))
            .one(db)
            .await
            .map_err(anyhow::Error::from)
    }

    async fn reset(
        &self,
        db: &DatabaseTransaction,
        skynet: &Skynet,
        uid: &HyUuid,
    ) -> Result<Option<users::Model>> {
        let password = utils::rand_string(16);
        let u = users::Entity::find_by_id(uid.to_owned()).one(db).await?;
        match u {
            Some(x) => {
                self.kick(skynet, uid).await?;
                let mut x: users::ActiveModel = x.into();
                x.password = Set(Self::hash_pass(
                    &password,
                    x.salt_prefix.as_ref(),
                    x.salt_suffix.as_ref(),
                ));
                let mut x = x.update(db).await?;
                x.password = password;
                Ok(Some(x))
            }
            None => Ok(None),
        }
    }

    async fn kick(&self, skynet: &Skynet, uid: &HyUuid) -> Result<()> {
        let mut redis = skynet.redis.clone().unwrap();
        let keys: Vec<String> = redis
            .keys(format!("{}*_{}", skynet.config.session_prefix.get(), uid))
            .await?;
        let mut re = redis::pipe();
        let mut re = re.atomic();
        for i in keys {
            re = re.del(i);
        }
        re.query_async(&mut redis)
            .await
            .map_err(anyhow::Error::from)
    }

    async fn delete_all(&self, db: &sea_orm::DatabaseTransaction, skynet: &Skynet) -> Result<u64> {
        let mut redis = skynet.redis.clone().unwrap();
        let keys: Vec<String> = redis
            .keys(format!("{}*_", skynet.config.session_prefix.get()))
            .await?;
        let mut re = redis::pipe();
        let mut re = re.atomic();
        for i in keys {
            re = re.del(i);
        }
        re.query_async(&mut redis).await?;
        users::Entity::delete_many()
            .exec(db)
            .await
            .map(|x| x.rows_affected)
            .map_err(anyhow::Error::from)
    }

    async fn delete(
        &self,
        db: &sea_orm::DatabaseTransaction,
        skynet: &Skynet,
        uid: &[skynet::HyUuid],
    ) -> Result<u64> {
        if uid.is_empty() {
            return Ok(0);
        }
        let mut redis = skynet.redis.clone().unwrap();
        let mut keys: Vec<String> = Vec::new();
        for i in uid {
            keys.append(
                &mut redis
                    .keys(format!("{}*_{}", skynet.config.session_prefix.get(), i))
                    .await?,
            );
        }
        let mut re = redis::pipe();
        let mut re = re.atomic();
        for i in keys {
            re = re.del(i);
        }
        re.query_async(&mut redis).await?;
        users::Entity::delete_many()
            .filter(users::Column::Id.is_in(uuid2string(uid)))
            .exec(db)
            .await
            .map(|x| x.rows_affected)
            .map_err(anyhow::Error::from)
    }
}
