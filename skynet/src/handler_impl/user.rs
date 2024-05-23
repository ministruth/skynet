use anyhow::{anyhow, Result};
use argon2::{password_hash::SaltString, Argon2, PasswordHash, PasswordHasher, PasswordVerifier};
use async_trait::async_trait;
use chrono::Utc;
use derivative::Derivative;
use rand::rngs::OsRng;
use redis::{aio::ConnectionManager, AsyncCommands};
use sea_orm::{
    ActiveModelTrait, ActiveValue::NotSet, ColumnTrait, DatabaseTransaction, EntityTrait,
    PaginatorTrait, QueryFilter, Set, Unchanged,
};
use skynet::{entity::users, handler::UserHandler, hyuuid::uuid2string, utils, HyUuid, Skynet};
use skynet_macro::default_handler_impl;

#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct DefaultUserHandler;

impl DefaultUserHandler {
    fn hash_pass(pass: &str) -> Result<String> {
        let argon2 = Argon2::default();
        let salt = SaltString::generate(&mut OsRng);
        Ok(argon2
            .hash_password(pass.as_bytes(), &salt)
            .map_err(|e| anyhow!(e))?
            .to_string())
    }
}

#[default_handler_impl(users)]
#[async_trait]
impl UserHandler for DefaultUserHandler {
    async fn create(
        &self,
        db: &DatabaseTransaction,
        username: &str,
        password: Option<&str>,
        avatar: Option<Vec<u8>>,
    ) -> Result<users::Model> {
        let password = password.map_or_else(|| utils::rand_string(32), ToOwned::to_owned);
        let mut user = users::ActiveModel {
            username: Set(username.to_owned()),
            password: Set(Self::hash_pass(&password)?),
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
        redis: &ConnectionManager,
        skynet: &Skynet,
        uid: &HyUuid,
        username: Option<&str>,
        password: Option<&str>,
        avatar: Option<Vec<u8>>,
    ) -> Result<users::Model> {
        let password = match password {
            Some(x) => {
                self.kick(redis, skynet, uid).await?;
                Set(Self::hash_pass(x)?)
            }
            None => NotSet,
        };
        users::ActiveModel {
            id: Unchanged(uid.to_owned()),
            username: username.map_or(NotSet, |x| Set(x.to_owned())),
            avatar: avatar.map_or(NotSet, |x| {
                if x.is_empty() {
                    Set(None)
                } else {
                    Set(Some(x))
                }
            }),
            password,
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
        users::ActiveModel {
            id: Unchanged(uid.to_owned()),
            last_ip: Set(Some(ip.to_owned())),
            last_login: Set(Some(Utc::now().timestamp_millis())),
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
                let hash = PasswordHash::new(&user.password).map_err(|e| anyhow!(e))?;
                if Argon2::default()
                    .verify_password(password.as_bytes(), &hash)
                    .is_ok()
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
        redis: &ConnectionManager,
        skynet: &Skynet,
        uid: &HyUuid,
    ) -> Result<Option<users::Model>> {
        let password = utils::rand_string(32);
        let u = users::Entity::find_by_id(uid.to_owned()).one(db).await?;
        match u {
            Some(x) => {
                self.kick(redis, skynet, uid).await?;
                let mut x: users::ActiveModel = x.into();
                x.password = Set(Self::hash_pass(&password)?);
                let mut x = x.update(db).await?;
                x.password = password;
                Ok(Some(x))
            }
            None => Ok(None),
        }
    }

    async fn kick(&self, redis: &ConnectionManager, skynet: &Skynet, uid: &HyUuid) -> Result<()> {
        let mut redis = redis.clone();
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

    async fn delete_all(
        &self,
        db: &sea_orm::DatabaseTransaction,
        redis: &ConnectionManager,
        skynet: &Skynet,
    ) -> Result<u64> {
        let mut redis = redis.clone();
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
        redis: &ConnectionManager,
        skynet: &Skynet,
        uid: &[skynet::HyUuid],
    ) -> Result<u64> {
        if uid.is_empty() {
            return Ok(0);
        }
        let mut redis = redis.clone();
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
