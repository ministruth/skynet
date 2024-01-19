//! Pluggable storage backends for session state.

mod interface;
mod session_key;

pub use self::{
    interface::{LoadError, SaveError, SessionStore, UpdateError},
    session_key::SessionKey,
};

mod redis_rs;
mod utils;
pub use redis_rs::{RedisSessionStore, RedisSessionStoreBuilder};
