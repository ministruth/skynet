use std::sync::Arc;

use actix_web::cookie::time::Duration;
use anyhow::Error;
use redis::{aio::ConnectionManager, AsyncCommands, Cmd, FromRedisValue, RedisResult, Value};

use super::SessionKey;
use crate::storage::{
    interface::{LoadError, SaveError, SessionState, UpdateError},
    utils::generate_session_key,
    SessionStore,
};

/// Use Redis as session storage backend.
///
/// ```no_run
/// use actix_web::{web, App, HttpServer, HttpResponse, Error};
/// use actix_session::{SessionMiddleware, storage::RedisSessionStore};
/// use actix_web::cookie::Key;
///
/// // The secret key would usually be read from a configuration file/environment variables.
/// fn get_secret_key() -> Key {
///     # todo!()
///     // [...]
/// }
///
/// #[actix_web::main]
/// async fn main() -> std::io::Result<()> {
///     let secret_key = get_secret_key();
///     let redis_connection_string = "redis://127.0.0.1:6379";
///     let store = RedisSessionStore::new(redis_connection_string).await.unwrap();
///
///     HttpServer::new(move ||
///             App::new()
///             .wrap(SessionMiddleware::new(
///                 store.clone(),
///                 secret_key.clone()
///             ))
///             .default_service(web::to(|| HttpResponse::Ok())))
///         .bind(("127.0.0.1", 8080))?
///         .run()
///         .await
/// }
/// ```
///
/// # TLS support
/// Add the `redis-rs-tls-session` feature flag to enable TLS support. You can then establish a TLS
/// connection to Redis using the `rediss://` URL scheme:
///
/// ```no_run
/// use actix_session::{storage::RedisSessionStore};
///
/// # actix_web::rt::System::new().block_on(async {
/// let redis_connection_string = "rediss://127.0.0.1:6379";
/// let store = RedisSessionStore::new(redis_connection_string).await.unwrap();
/// # })
/// ```
///
/// # Implementation notes
/// `RedisSessionStore` leverages [`redis-rs`] as Redis client.
///
/// [`redis-rs`]: https://github.com/mitsuhiko/redis-rs
#[derive(Clone)]
pub struct RedisSessionStore {
    configuration: CacheConfiguration,
    client: ConnectionManager,
}

#[derive(Clone)]
struct CacheConfiguration {
    cache_keygen: Arc<dyn Fn(&str) -> String + Send + Sync>,
}

impl Default for CacheConfiguration {
    fn default() -> Self {
        Self {
            cache_keygen: Arc::new(str::to_owned),
        }
    }
}

impl RedisSessionStore {
    /// A fluent API to configure [`RedisSessionStore`].
    /// It takes as input the only required input to create a new instance of [`RedisSessionStore`] - a
    /// connection string for Redis.
    pub fn builder(client: ConnectionManager) -> RedisSessionStoreBuilder {
        RedisSessionStoreBuilder {
            configuration: CacheConfiguration::default(),
            client,
        }
    }

    /// Create a new instance of [`RedisSessionStore`] using the default configuration.
    /// It takes as input the only required input to create a new instance of [`RedisSessionStore`] - a
    /// connection string for Redis.
    pub fn new(client: ConnectionManager) -> Result<RedisSessionStore, anyhow::Error> {
        Self::builder(client).build()
    }
}

/// A fluent builder to construct a [`RedisSessionStore`] instance with custom configuration
/// parameters.
///
/// [`RedisSessionStore`]: crate::storage::RedisSessionStore
#[must_use]
pub struct RedisSessionStoreBuilder {
    client: ConnectionManager,
    configuration: CacheConfiguration,
}

impl RedisSessionStoreBuilder {
    /// Set a custom cache key generation strategy, expecting a session key as input.
    pub fn cache_keygen<F>(mut self, keygen: F) -> Self
    where
        F: Fn(&str) -> String + 'static + Send + Sync,
    {
        self.configuration.cache_keygen = Arc::new(keygen);
        self
    }

    /// Finalise the builder and return a [`RedisSessionStore`] instance.
    pub fn build(self) -> Result<RedisSessionStore, anyhow::Error> {
        Ok(RedisSessionStore {
            configuration: self.configuration,
            client: self.client,
        })
    }
}

impl SessionStore for RedisSessionStore {
    async fn load(&self, session_key: &SessionKey) -> Result<Option<SessionState>, LoadError> {
        let mut cache_key = (self.configuration.cache_keygen)(session_key.as_ref());
        cache_key.push_str("_*");

        let key: Vec<String> = self
            .execute_command(redis::cmd("KEYS").arg(&[&cache_key]))
            .await
            .map_err(Into::into)
            .map_err(LoadError::Other)?;
        if key.len() > 1 {
            return Err(LoadError::Other(anyhow::anyhow!("session key conflict")));
        }

        if let Some(x) = key.first() {
            let value: Option<String> = self
                .execute_command(redis::cmd("GET").arg(&[x]))
                .await
                .map_err(Into::into)
                .map_err(LoadError::Other)?;

            match value {
                None => Ok(None),
                Some(value) => Ok(serde_json::from_str(&value)
                    .map_err(Into::into)
                    .map_err(LoadError::Deserialization)?),
            }
        } else {
            Ok(None)
        }
    }

    async fn save(
        &self,
        session_state: SessionState,
        ttl: &Duration,
    ) -> Result<SessionKey, SaveError> {
        let body = serde_json::to_string(&session_state)
            .map_err(Into::into)
            .map_err(SaveError::Serialization)?;
        let session_key = generate_session_key();
        let mut cache_key = (self.configuration.cache_keygen)(session_key.as_ref());
        let id = session_state
            .get("id")
            .ok_or(SaveError::Other(anyhow::anyhow!("id not in session")))?
            .trim_matches('"');
        cache_key.push_str(&format!("_{}", id));

        self.execute_command(redis::cmd("SET").arg(&[
            &cache_key,
            &body,
            "NX", // NX: only set the key if it does not already exist
            "EX", // EX: set expiry
            &format!("{}", ttl.whole_seconds()),
        ]))
        .await
        .map_err(Into::into)
        .map_err(SaveError::Other)?;

        Ok(session_key)
    }

    async fn update(
        &self,
        session_key: SessionKey,
        session_state: SessionState,
        ttl: &Duration,
    ) -> Result<SessionKey, UpdateError> {
        let body = serde_json::to_string(&session_state)
            .map_err(Into::into)
            .map_err(UpdateError::Serialization)?;

        let mut cache_key = (self.configuration.cache_keygen)(session_key.as_ref());
        let id = session_state
            .get("id")
            .ok_or(UpdateError::Other(anyhow::anyhow!("id not in session")))?
            .trim_matches('"');
        cache_key.push_str(&format!("_{}", id));

        let v: redis::Value = self
            .execute_command(redis::cmd("SET").arg(&[
                &cache_key,
                &body,
                "XX", // XX: Only set the key if it already exist.
                "EX", // EX: set expiry
                &format!("{}", ttl.whole_seconds()),
            ]))
            .await
            .map_err(Into::into)
            .map_err(UpdateError::Other)?;

        match v {
            Value::Nil => {
                // The SET operation was not performed because the XX condition was not verified.
                // This can happen if the session state expired between the load operation and the
                // update operation. Unlucky, to say the least. We fall back to the `save` routine
                // to ensure that the new key is unique.
                self.save(session_state, ttl)
                    .await
                    .map_err(|err| match err {
                        SaveError::Serialization(err) => UpdateError::Serialization(err),
                        SaveError::Other(err) => UpdateError::Other(err),
                    })
            }
            Value::Int(_) | Value::Okay | Value::Status(_) => Ok(session_key),
            val => Err(UpdateError::Other(anyhow::anyhow!(
                "Failed to update session state. {:?}",
                val
            ))),
        }
    }

    async fn update_ttl(&self, session_key: &SessionKey, ttl: &Duration) -> Result<(), Error> {
        let mut cache_key = (self.configuration.cache_keygen)(session_key.as_ref());
        cache_key.push_str("_*");
        let key: Vec<String> = self
            .execute_command(redis::cmd("KEYS").arg(&[&cache_key]))
            .await
            .map_err(Into::into)
            .map_err(UpdateError::Other)?;
        if key.len() > 1 {
            anyhow::bail!(UpdateError::Other(anyhow::anyhow!("session key conflict")));
        }

        if let Some(x) = key.first() {
            self.client.clone().expire(x, ttl.whole_seconds()).await?;
        }
        Ok(())
    }

    async fn delete(&self, session_key: &SessionKey) -> Result<(), anyhow::Error> {
        let mut cache_key = (self.configuration.cache_keygen)(session_key.as_ref());
        cache_key.push_str("_*");

        let key: Vec<String> = self
            .execute_command(redis::cmd("KEYS").arg(&[&cache_key]))
            .await
            .map_err(Into::into)
            .map_err(UpdateError::Other)?;
        if key.len() > 1 {
            anyhow::bail!(UpdateError::Other(anyhow::anyhow!("session key conflict")));
        }
        if let Some(x) = key.first() {
            self.execute_command(redis::cmd("DEL").arg(&[x]))
                .await
                .map_err(Into::into)
                .map_err(UpdateError::Other)?;
        }

        Ok(())
    }
}

impl RedisSessionStore {
    /// Execute Redis command and retry once in certain cases.
    ///
    /// `ConnectionManager` automatically reconnects when it encounters an error talking to Redis.
    /// The request that bumped into the error, though, fails.
    ///
    /// This is generally OK, but there is an unpleasant edge case: Redis client timeouts. The
    /// server is configured to drop connections who have been active longer than a pre-determined
    /// threshold. `redis-rs` does not proactively detect that the connection has been dropped - you
    /// only find out when you try to use it.
    ///
    /// This helper method catches this case (`.is_connection_dropped`) to execute a retry. The
    /// retry will be executed on a fresh connection, therefore it is likely to succeed (or fail for
    /// a different more meaningful reason).
    #[allow(clippy::needless_pass_by_ref_mut)]
    async fn execute_command<T: FromRedisValue>(&self, cmd: &mut Cmd) -> RedisResult<T> {
        let mut can_retry = true;

        loop {
            match cmd.query_async(&mut self.client.clone()).await {
                Ok(value) => return Ok(value),
                Err(err) => {
                    if can_retry && err.is_connection_dropped() {
                        tracing::debug!(
                            "Connection dropped while trying to talk to Redis. Retrying."
                        );

                        // Retry at most once
                        can_retry = false;

                        continue;
                    } else {
                        return Err(err);
                    }
                }
            }
        }
    }
}
