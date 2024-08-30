use std::{
    path::PathBuf,
    sync::{
        atomic::{AtomicU64, Ordering},
        Arc, OnceLock,
    },
    thread,
};

use parking_lot::Mutex;
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use skynet_api::{
    actix_cloud::{
        logger::{LogItem, LoggerBuilder, LoggerGuard},
        tokio::runtime,
        tracing::Level,
    },
    entity::notifications,
    logger::Logger,
    request::NotifyLevel,
    sea_orm::{ActiveModelTrait, DatabaseConnection, Set},
};

static DB: OnceLock<DatabaseConnection> = OnceLock::new();
static CACHE: OnceLock<Mutex<Vec<notifications::ActiveModel>>> = OnceLock::new();
static UNREAD: OnceLock<Arc<AtomicU64>> = OnceLock::new();

#[derive(Deserialize, Serialize, Debug)]
struct DetailItem {
    #[serde(flatten)]
    fields: Map<String, Value>,
    #[serde(flatten)]
    span: Map<String, Value>,
}

/// Initialize db connection. Cached data will be written.
///
/// # Panics
///
/// Panics if the database connection is already set.
pub async fn set_db(db: DatabaseConnection) {
    DB.set(db).unwrap();
    write_pending(DB.get().unwrap()).await
}

/// Write a notification to the database.
/// When `success` is `true`, `level` is ignored and a special level `Success` is set.
async fn write_notification(
    success: bool,
    level: Level,
    target: String,
    msg: String,
    detail: String,
) {
    let level = if success {
        NotifyLevel::Success
    } else if level == Level::ERROR {
        NotifyLevel::Error
    } else {
        NotifyLevel::Warning
    };
    let model = notifications::ActiveModel {
        level: Set(level as i32),
        target: Set(target),
        message: Set(msg),
        detail: Set(detail),
        ..Default::default()
    };
    if let Some(db) = DB.get() {
        write_pending(db).await;
        let _ = model.insert(db).await;
    } else {
        CACHE.get().unwrap().lock().push(model);
    }
    if !success {
        UNREAD.get().unwrap().fetch_add(1, Ordering::SeqCst);
    }
}

/// Write pending notifications to database.
///
/// # Panics
///
/// Panics if cache is not initialized.
async fn write_pending(db: &DatabaseConnection) {
    let cached: Vec<_> = CACHE.get().unwrap().lock().drain(..).collect();
    for i in cached {
        let _ = i.insert(db).await;
    }
}

fn filter(item: &LogItem) -> bool {
    // ignore lots of https log
    if item.target.starts_with("h2::") && item.level > Level::INFO {
        return false;
    }
    // ignore https suite log
    if item.target.starts_with("rustls::server::hs") && item.level > Level::INFO {
        return false;
    }
    // ignore middleware log, use our own
    if item.target.starts_with("tracing_actix_web::middleware")
        || item.target.starts_with("actix_files::service")
        || item.target.starts_with("actix_http::h1")
        || item.target.starts_with("actix_web_validator::")
    {
        return false;
    }
    true
}

fn transformer(mut item: LogItem) -> LogItem {
    // Trim target path.
    // Only keep the first path before `::`. Change several library targets to our own.
    if item.target.starts_with("actix_") {
        item.target = String::from("skynet-web");
    } else if item.target.starts_with("sea_orm::") || item.target.starts_with("sea_orm_migration::")
    {
        item.target = String::from("skynet-db");
    } else {
        item.target = item
            .target
            .split("::")
            .next()
            .unwrap_or("unknown")
            .to_owned();
    }

    // Remove useless file path.
    // Only keep path starting from the last `src`.
    if let Some(filename) = &item.filename {
        let mut buf = Vec::new();
        let mut flag = false;
        let s = PathBuf::from(filename);
        for i in s.iter().rev() {
            buf.push(i);
            if flag {
                break;
            }
            if i.eq_ignore_ascii_case("src") {
                flag = true;
            }
        }
        let mut s = PathBuf::new();
        for i in buf.into_iter().rev() {
            s.push(i);
        }
        item.filename = Some(s.to_string_lossy().into());
    }

    let success = item
        .fields
        .get("success")
        .and_then(Value::as_bool)
        .unwrap_or_default();
    if item.level <= Level::WARN || success {
        // prevent deadlock for sqlite
        let level = item.level;
        let target = item.target.clone();
        let message = item.message.clone();
        let mut detail = DetailItem {
            fields: item.fields.clone(),
            span: item.span.clone(),
        };
        if let Some(filename) = &item.filename {
            detail
                .fields
                .insert("filename".into(), filename.to_owned().into());
        }
        if let Some(line_number) = &item.line_number {
            detail
                .fields
                .insert("line_number".into(), line_number.to_owned().into());
        }
        thread::spawn(move || {
            runtime::Builder::new_current_thread()
                .enable_time()
                .build()
                .unwrap()
                .block_on(async {
                    write_notification(
                        success,
                        level,
                        target,
                        message,
                        serde_json::to_string(&detail).unwrap_or_default(),
                    )
                    .await;
                });
        });
    }
    item
}

pub fn start_logger(enable: bool, json: bool, verbose: bool) -> (Logger, Option<LoggerGuard>) {
    let unread = Arc::new(AtomicU64::new(0));
    CACHE.set(Mutex::new(Vec::new())).unwrap();
    UNREAD.set(unread.clone()).unwrap();
    let (logger, guard) = if enable {
        let mut builder = LoggerBuilder::new();
        if json {
            builder = builder.json();
        }
        if verbose {
            builder = builder.filename().line_number().level(Level::DEBUG);
        }
        builder = builder.transformer(transformer).filter(filter);
        let (logger, guard) = builder.start();
        (Some(logger), Some(guard))
    } else {
        (None, None)
    };

    (
        Logger {
            verbose,
            json,
            enable,
            unread,
            logger,
        },
        guard,
    )
}
