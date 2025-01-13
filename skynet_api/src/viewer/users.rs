use std::str::FromStr;

use crate::{
    entity::{user_histories, users},
    hyuuid::uuids2strings,
    permission::ROOT_ID,
    request::{Condition, Session},
    HyUuid,
};
use actix_cloud::{memorydb::MemoryDB, utils};
use anyhow::{anyhow, Result};
use argon2::{
    password_hash::{rand_core::OsRng, SaltString},
    Argon2, PasswordHash, PasswordHasher, PasswordVerifier,
};
use chrono::Utc;
use sea_orm::{
    ActiveModelTrait, ActiveValue::NotSet, ColumnTrait, ConnectionTrait, DatabaseTransaction,
    EntityTrait, PaginatorTrait, QueryFilter, Set, Unchanged,
};
use skynet_macro::default_viewer;

pub struct UserViewer;

#[default_viewer(users)]
impl UserViewer {
    fn hash_pass(pass: &str) -> Result<String> {
        let argon2 = Argon2::default();
        let salt = SaltString::generate(&mut OsRng);
        Ok(argon2
            .hash_password(pass.as_bytes(), &salt)
            .map_err(|e| anyhow!(e))?
            .to_string())
    }

    /// Create new user, generate random password if not set.
    /// Returned password is in plain text.
    pub async fn create<C>(
        db: &C,
        username: &str,
        password: Option<&str>,
        avatar: Option<Vec<u8>>,
        root: bool,
    ) -> Result<users::Model>
    where
        C: ConnectionTrait,
    {
        let password = password
            .map(ToOwned::to_owned)
            .unwrap_or_else(|| utils::rand_string(32));
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

    /// Update login `uid` and `ip`.
    pub async fn update_login(
        db: &DatabaseTransaction,
        uid: &HyUuid,
        ip: &str,
        user_agent: Option<&str>,
    ) -> Result<users::Model> {
        let ts = Utc::now().timestamp_millis();
        user_histories::ActiveModel {
            uid: Set(*uid),
            ip: Set(ip.to_owned()),
            user_agent: Set(user_agent.map(|x| x.into())),
            created_at: Set(ts),
            updated_at: Set(ts),
            ..Default::default()
        }
        .insert(db)
        .await?;
        users::ActiveModel {
            id: Unchanged(*uid),
            last_ip: Set(Some(ip.to_owned())),
            last_login: Set(Some(ts)),
            ..Default::default()
        }
        .update(db)
        .await
        .map_err(Into::into)
    }

    /// Find user by `username`.
    /// Return `None` when not found.
    pub async fn find_by_name<C>(db: &C, username: &str) -> Result<Option<users::Model>>
    where
        C: ConnectionTrait,
    {
        users::Entity::find()
            .filter(users::Column::Username.eq(username))
            .one(db)
            .await
            .map_err(Into::into)
    }

    /// Reset user password by `uid`.
    /// Return `None` when not found, password is in plain text otherwise.
    ///
    /// User will be kicked after reset.
    pub async fn reset<M>(
        db: &DatabaseTransaction,
        memorydb: &M,
        uid: &HyUuid,
        session_prefix: &str,
    ) -> Result<Option<users::Model>>
    where
        M: MemoryDB + ?Sized,
    {
        let password = utils::rand_string(32);
        let u = users::Entity::find_by_id(*uid).one(db).await?;
        match u {
            Some(x) => {
                let mut x: users::ActiveModel = x.into();
                x.password = Set(Self::hash_pass(&password)?);
                let mut x = x.update(db).await?;
                x.password = password;
                Self::kick(memorydb, uid, session_prefix).await?;
                Ok(Some(x))
            }
            None => Ok(None),
        }
    }

    /// Kick all `uid` login sessions.
    pub async fn kick<M>(db: &M, uid: &HyUuid, session_prefix: &str) -> Result<()>
    where
        M: MemoryDB + ?Sized,
    {
        let s = db.keys(&format!("{}{}_*", session_prefix, uid)).await?;
        let prefix = format!("{}{}_", session_prefix, uid);
        let mut keys = Vec::new();
        for i in s.iter() {
            keys.push(i.replace(&prefix, session_prefix));
        }
        for i in s {
            keys.push(i);
        }
        db.dels(&keys).await?;
        Ok(())
    }

    /// Check `username` `password`.
    ///
    /// - If error, return `Err`.
    /// - If `username` not found, return `(false, None)`.
    /// - If `password` not match, return `(false, Some)`.
    /// - If success, return `(true, Some)`.
    pub async fn check_pass<C>(
        db: &C,
        username: &str,
        password: &str,
    ) -> Result<(bool, Option<users::Model>)>
    where
        C: ConnectionTrait,
    {
        let user = Self::find_by_name(db, username).await?;
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

    /// Update user infos by `uid`.
    /// Password will be automatically hashed.
    ///
    /// User will be kicked if password is updated.
    pub async fn update<M>(
        db: &DatabaseTransaction,
        memorydb: &M,
        uid: &HyUuid,
        username: Option<&str>,
        password: Option<&str>,
        avatar: Option<Vec<u8>>,
        session_prefix: &str,
    ) -> Result<users::Model>
    where
        M: MemoryDB + ?Sized,
    {
        let ret = users::ActiveModel {
            id: Unchanged(*uid),
            username: username.map_or(NotSet, |x| Set(x.to_owned())),
            avatar: avatar.map_or(NotSet, |x| {
                if x.is_empty() {
                    Set(None)
                } else {
                    Set(Some(x))
                }
            }),
            password: match &password {
                Some(x) => Set(Self::hash_pass(x)?),
                None => NotSet,
            },
            ..Default::default()
        }
        .update(db)
        .await?;
        if password.is_some() {
            Self::kick(memorydb, uid, session_prefix).await?;
        }
        Ok(ret)
    }

    /// Delete all users and kick them.
    pub async fn delete_all<M>(
        db: &DatabaseTransaction,
        memorydb: &M,
        session_prefix: &str,
    ) -> Result<u64>
    where
        M: MemoryDB + ?Sized,
    {
        let ret = users::Entity::delete_many()
            .exec(db)
            .await
            .map(|x| x.rows_affected)?;
        let s = memorydb.keys(&format!("{}*", session_prefix)).await?;
        memorydb.dels(&s).await?;
        Ok(ret)
    }

    /// Delete `uid` users and kick them.
    pub async fn delete<M>(
        db: &DatabaseTransaction,
        memorydb: &M,
        uid: &[HyUuid],
        session_prefix: &str,
    ) -> Result<u64>
    where
        M: MemoryDB + ?Sized,
    {
        if uid.is_empty() {
            return Ok(0);
        }
        let cnt = users::Entity::delete_many()
            .filter(users::Column::Id.is_in(uuids2strings(uid)))
            .exec(db)
            .await
            .map(|x| x.rows_affected)?;
        for i in uid {
            Self::kick(memorydb, i, session_prefix).await?;
        }
        Ok(cnt)
    }

    pub async fn find_history_by_id<C>(
        db: &C,
        id: &HyUuid,
        cond: Condition,
    ) -> Result<(Vec<user_histories::Model>, u64)>
    where
        C: ConnectionTrait,
    {
        cond.select_page(
            user_histories::Entity::find().filter(user_histories::Column::Uid.eq(*id)),
            db,
        )
        .await
    }

    pub async fn find_sessions<M>(
        db: &M,
        uid: &HyUuid,
        session_prefix: &str,
    ) -> Result<Vec<Session>>
    where
        M: MemoryDB + ?Sized,
    {
        let s = db.keys(&format!("{}{}_*", session_prefix, uid)).await?;
        let prefix = format!("{}{}_", session_prefix, uid);
        let mut keys = Vec::new();
        for i in s.iter() {
            keys.push(i.replace(&prefix, session_prefix));
        }
        let mut sessions = Vec::new();
        for i in keys {
            if let Some(x) = db.get(&i).await? {
                let s = Session::from_str(&x)?;
                sessions.push(s);
            }
        }
        Ok(sessions)
    }
}
