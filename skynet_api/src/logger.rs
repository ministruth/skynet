use std::sync::{
    atomic::{AtomicU64, Ordering},
    Arc,
};

use actix_cloud::{logger, tracing::Level};
use serde_repr::{Deserialize_repr, Serialize_repr};

#[derive(Serialize_repr, Deserialize_repr, Debug, PartialEq, Eq, Hash, Clone, Copy)]
#[repr(i32)]
pub enum NotifyLevel {
    Info = 0,
    Success,
    Warning,
    Error,
}

pub struct Logger {
    pub verbose: bool,
    pub json: bool,
    pub enable: bool,
    pub unread: Arc<AtomicU64>,
    pub logger: Option<logger::Logger>,
}

impl Logger {
    fn get_builder(&self) -> logger::LoggerBuilder {
        let mut logger = logger::LoggerBuilder::new();
        if self.verbose {
            logger = logger.filename().line_number().level(Level::DEBUG);
        }
        logger
    }

    /// Set unread notifications to `num`.
    pub fn set_unread(&self, num: u64) {
        self.unread.store(num, Ordering::SeqCst);
    }

    /// Add `num` to unread notifications.
    pub fn add_unread(&self, num: u64) {
        self.unread.fetch_add(num, Ordering::SeqCst);
    }

    /// Get the number of unread notifications.
    pub fn get_unread(&self) -> u64 {
        self.unread.load(Ordering::Relaxed)
    }

    /// Init logger, plugins will call this automatically when loaded.
    pub fn init(&self) {
        if let Some(logger) = &self.logger {
            logger.init(&self.get_builder())
        }
    }
}
