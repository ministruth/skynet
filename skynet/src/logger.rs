use std::{
    io,
    sync::{atomic::Ordering, OnceLock},
    time,
};

use actix_web::rt::task;
use anyhow::Result;
use derivative::Derivative;
use fern::colors::{Color, ColoredLevelConfig};
use futures::executor::block_on;
use log::{Level, LevelFilter};
use parking_lot::Mutex;
use sea_orm::{ActiveModelTrait, DatabaseConnection, Set};
use serde_json::json;

use crate::{entity::notifications, NotifyLevel, UNREAD_NOTIFICATIONS};

pub static DBLOGGER: OnceLock<DatabaseConnection> = OnceLock::new();
static DBCACHE: OnceLock<Mutex<Vec<(NotifyLevel, String, String)>>> = OnceLock::new();

#[must_use]
#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct Logger {
    pub verbose: bool,
    pub json: bool,
    pub enable: bool,
}

impl Logger {
    /// Normalize target to skynet target.
    #[must_use]
    fn normalize_target(s: &str) -> String {
        if s.starts_with("skynet::") || s == "_success" {
            String::from("skynet")
        } else if s.starts_with("actix_") {
            String::from("skynet-web")
        } else if s.starts_with("sea_orm::") || s.starts_with("sea_orm_migration::") {
            String::from("skynet-db")
        } else {
            s.to_owned()
        }
    }

    fn write_notification(success: bool, level: log::Level, target: String, mut msg: String) {
        if level == Level::Warn || level == Level::Error || success {
            let level = if success {
                NotifyLevel::Success
            } else if level == Level::Error {
                NotifyLevel::Error
            } else {
                NotifyLevel::Warning
            };
            if let Some(db) = DBLOGGER.get() {
                // prevent deadlock
                let split = msg
                    .split_once('\n')
                    .map(|x| (x.0.to_owned(), x.1.to_owned()));
                let detail = match split {
                    Some(x) => {
                        msg = x.0;
                        x.1
                    }
                    None => String::new(),
                };
                task::spawn_blocking(move || {
                    block_on(async {
                        let _ = notifications::ActiveModel {
                            level: Set(level as i32),
                            target: Set(target),
                            message: Set(msg),
                            detail: Set(detail),
                            ..Default::default()
                        }
                        .insert(db)
                        .await
                        .unwrap();
                    });
                });
            } else {
                DBCACHE.get().unwrap().lock().push((level, target, msg));
            }
            UNREAD_NOTIFICATIONS.fetch_add(1, Ordering::SeqCst);
        }
    }

    /// Reinit the logger.
    /// # Warning
    ///
    /// Do NOT call this function manually.
    ///
    /// # Errors
    ///
    /// Will return `Err` if logger cannot be set.
    pub fn reinit(&mut self) -> Result<()> {
        self.init(self.enable, self.json, self.verbose)
    }

    /// Write pending warnings and errors to database.
    /// The pending ones will be removed after this function.
    ///
    /// # Panics
    ///
    /// Panics if the logger is not initialized.
    pub async fn write_db_pending(db: &DatabaseConnection) {
        let cached: Vec<(i32, String, String)> = DBCACHE
            .get()
            .unwrap()
            .lock()
            .drain(..)
            .map(|x| (x.0 as i32, x.1, x.2))
            .collect();
        if !cached.is_empty() {
            for i in cached {
                let _ = notifications::ActiveModel {
                    level: Set(i.0),
                    target: Set(i.1),
                    message: Set(i.2),
                    detail: Set(String::new()),
                    ..Default::default()
                }
                .insert(db)
                .await;
            }
        }
    }

    /// Init the logger.
    ///
    /// # Warning
    ///
    /// Do NOT call this function manually.
    ///
    /// # Errors
    ///
    /// Will return `Err` if logger cannot be set.
    #[allow(clippy::missing_panics_doc)]
    pub fn init(&mut self, enable: bool, json: bool, verbose: bool) -> Result<()> {
        if enable {
            DBCACHE.set(Mutex::new(Vec::new())).unwrap();
            let level_color = ColoredLevelConfig::new()
                .debug(Color::BrightMagenta)
                .info(Color::BrightBlue)
                .warn(Color::BrightYellow)
                .error(Color::BrightRed);
            let mut logger = fern::Dispatch::new()
                .level_for("rustls::server::hs", LevelFilter::Info) // ignore https suite log
                .level_for("h2", LevelFilter::Info) // ignore lots of https log
                .level_for("actix_web::middleware::logger", LevelFilter::Info) // disable debug web log, use our own
                .level(if verbose {
                    LevelFilter::Debug
                } else {
                    LevelFilter::Info
                });
            logger = logger.format(move |out, message, record| {
                let target = Self::normalize_target(record.target());
                let message = message.to_string();
                Self::write_notification(
                    record.target() == "_success",
                    record.level(),
                    target.clone(),
                    message.clone(),
                );
                let message = message.replace('\n', "\t");
                if json {
                    let time = time::SystemTime::now()
                        .duration_since(time::UNIX_EPOCH)
                        .unwrap()
                        .as_secs();
                    out.finish(format_args!(
                        "{}",
                        serde_json::to_string(&json!({
                            "time":time,
                            "target":target,
                            "level":record.level().as_str(),
                            "msg":message,
                        }))
                        .unwrap()
                    ));
                } else {
                    out.finish(format_args!(
                        "{}[{}][{}] {}",
                        chrono::Local::now().format("[%Y-%m-%d %H:%M:%S]"),
                        target,
                        level_color.color(record.level()),
                        message
                    ));
                }
            });

            logger = logger
                .chain(
                    fern::Dispatch::new()
                        .filter(|s| s.level() != LevelFilter::Error)
                        .chain(io::stdout()),
                )
                .chain(
                    fern::Dispatch::new()
                        .level(LevelFilter::Error)
                        .chain(io::stderr()),
                );
            logger.apply()?;
        }
        self.json = json;
        self.verbose = verbose;
        self.enable = enable;
        Ok(())
    }
}
