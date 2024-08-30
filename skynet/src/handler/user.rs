use std::sync::Arc;

use argon2::{password_hash::SaltString, Argon2, PasswordHash, PasswordHasher, PasswordVerifier};
use derivative::Derivative;
use rand::rngs::OsRng;
use skynet_api::{
    actix_cloud::{memorydb::MemoryDB, utils},
    anyhow, async_trait,
    entity::users,
    handler::UserHandler,
    hyuuid::uuids2strings,
    permission::ROOT_ID,
    sea_orm::{
        sqlx::types::chrono::Utc, ActiveModelTrait, ActiveValue::NotSet, ColumnTrait,
        DatabaseTransaction, EntityTrait, PaginatorTrait, QueryFilter, Set, Unchanged,
    },
    HyUuid, Result, Skynet,
};
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
        root: bool,
    ) -> Result<users::Model> {
        let password = password.map_or_else(|| utils::rand_string(32), ToOwned::to_owned);
        let mut user = users::ActiveModel {
            username: Set(username.to_owned()),
            password: Set(Self::hash_pass(&password)?),
            avatar: Set(avatar),
            ..Default::default()
        };
        if root {
            user.id = Set(ROOT_ID);
        }
        let mut user = user.insert(db).await?;
        user.password = password;
        Ok(user)
    }

    async fn update(
        &self,
        db: &DatabaseTransaction,
        memorydb: Arc<dyn MemoryDB>,
        skynet: &Skynet,
        uid: &HyUuid,
        username: Option<&str>,
        password: Option<&str>,
        avatar: Option<Vec<u8>>,
    ) -> Result<users::Model> {
        let ret = users::ActiveModel {
            id: Unchanged(uid.to_owned()),
            username: username.map_or(NotSet, |x| Set(x.to_owned())),
            avatar: avatar.map_or(NotSet, |x| {
                if x.is_empty() {
                    Set(None)
                } else {
                    Set(Some(x))
                }
            }),
            password: match password {
                Some(x) => Set(Self::hash_pass(x)?),
                None => NotSet,
            },
            ..Default::default()
        }
        .update(db)
        .await?;
        self.kick(memorydb, skynet, uid).await?;
        Ok(ret)
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
        .map_err(Into::into)
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
            .map_err(Into::into)
    }

    async fn reset(
        &self,
        db: &DatabaseTransaction,
        memorydb: Arc<dyn MemoryDB>,
        skynet: &Skynet,
        uid: &HyUuid,
    ) -> Result<Option<users::Model>> {
        let password = utils::rand_string(32);
        let u = users::Entity::find_by_id(uid.to_owned()).one(db).await?;
        match u {
            Some(x) => {
                let mut x: users::ActiveModel = x.into();
                x.password = Set(Self::hash_pass(&password)?);
                let mut x = x.update(db).await?;
                x.password = password;
                self.kick(memorydb, skynet, uid).await?;
                Ok(Some(x))
            }
            None => Ok(None),
        }
    }

    async fn kick(&self, memorydb: Arc<dyn MemoryDB>, skynet: &Skynet, uid: &HyUuid) -> Result<()> {
        let s = memorydb
            .keys(&format!("{}{}_*", skynet.config.session.prefix, uid))
            .await?;
        let prefix = format!("{}{}_", skynet.config.session.prefix, uid);
        let mut keys = Vec::new();
        for i in s.iter() {
            keys.push(i.replace(&prefix, &skynet.config.session.prefix));
        }
        for i in s {
            keys.push(i);
        }
        memorydb.dels(&keys).await?;
        Ok(())
    }

    async fn delete_all(
        &self,
        db: &DatabaseTransaction,
        memorydb: Arc<dyn MemoryDB>,
        skynet: &Skynet,
    ) -> Result<u64> {
        let ret = users::Entity::delete_many()
            .exec(db)
            .await
            .map(|x| x.rows_affected)?;
        let s = memorydb
            .keys(&format!("{}*", skynet.config.session.prefix))
            .await?;
        memorydb.dels(&s).await?;
        Ok(ret)
    }

    async fn delete(
        &self,
        db: &DatabaseTransaction,
        memorydb: Arc<dyn MemoryDB>,
        skynet: &Skynet,
        uid: &[HyUuid],
    ) -> Result<u64> {
        if uid.is_empty() {
            return Ok(0);
        }
        let cnt = users::Entity::delete_many()
            .filter(users::Column::Id.is_in(uuids2strings(uid)))
            .exec(db)
            .await
            .map(|x| x.rows_affected)?;
        for i in uid {
            self.kick(memorydb.clone(), skynet, i).await?;
        }
        Ok(cnt)
    }
}
