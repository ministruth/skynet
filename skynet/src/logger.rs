use anyhow::Result;
use chrono::{DateTime, Local, Utc};
use colored::{Color, Colorize};
use derivative::Derivative;
use parking_lot::Mutex;
use sea_orm::{ActiveModelTrait, DatabaseConnection, Set};
use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};
use std::{
    fmt::Write as _,
    io::{self, stderr, stdout, Write},
    path::PathBuf,
    str::FromStr,
    sync::{
        atomic::{AtomicU64, Ordering},
        mpsc::{self, Sender},
        Arc, OnceLock,
    },
    thread,
};
use tokio::runtime;
use tracing::Level;

use crate::{entity::notifications, utils, NotifyLevel};

static CACHE: OnceLock<Mutex<Vec<notifications::ActiveModel>>> = OnceLock::new();
static DB: OnceLock<DatabaseConnection> = OnceLock::new();

#[derive(Serialize, Debug)]
pub struct LogItem {
    pub time: Value,
    pub level: String,
    pub message: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub target: String,
    #[serde(skip_serializing_if = "Map::is_empty")]
    pub fields: Map<String, Value>,
    #[serde(skip_serializing_if = "Map::is_empty")]
    pub span: Map<String, Value>,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub filename: String,
    #[serde(skip_serializing_if = "utils::is_default")]
    pub line_number: i64,
}

impl LogItem {
    fn json_take_object(mp: &mut Map<String, Value>, key: &str) -> Map<String, Value> {
        if let Value::Object(x) = mp.remove(key).unwrap_or_default() {
            x
        } else {
            Map::default()
        }
    }

    #[must_use]
    pub fn from_json(mut s: Map<String, Value>) -> Self {
        let target = s
            .get("target")
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_owned();
        let level = s
            .get("level")
            .and_then(Value::as_str)
            .unwrap_or("ERROR")
            .to_owned();
        let filename = s
            .get("filename")
            .and_then(Value::as_str)
            .unwrap_or_default()
            .to_owned();
        let line_number = s
            .get("line_number")
            .and_then(Value::as_i64)
            .unwrap_or_default();
        let mut fields = Self::json_take_object(&mut s, "fields");
        let message = fields
            .remove("message")
            .unwrap_or_default()
            .as_str()
            .unwrap_or_default()
            .to_owned();
        let mut span = Self::json_take_object(&mut s, "span");
        span.remove("name");
        Self {
            time: Value::default(),
            level,
            message,
            target,
            fields,
            span,
            filename,
            line_number,
        }
    }

    /// # Errors
    ///
    /// Will raise `Err` for write errors.
    pub fn write_json<T: Write>(&self, mut writer: T) -> Result<()> {
        let v = serde_json::to_string(self).unwrap_or_default();
        writer.write_fmt(format_args!("{v}\n")).map_err(Into::into)
    }

    /// # Errors
    ///
    /// Will raise `Err` for write errors.
    pub fn write_console<T: Write>(&self, mut writer: T) -> Result<()> {
        let mut buf = String::new();
        write!(
            buf,
            "{} {} ",
            self.time.as_str().unwrap_or_default().bright_black(),
            self.level
        )?;
        if !self.span.is_empty() {
            write!(
                buf,
                "{}({}) ",
                &self
                    .span
                    .get("request_id")
                    .and_then(Value::as_str)
                    .unwrap_or("deadbeef")[..8],
                self.span
                    .get("ip")
                    .and_then(Value::as_str)
                    .unwrap_or("0.0.0.0")
            )?;
        }
        if !self.target.is_empty() {
            write!(buf, "{}", self.target.bright_black())?;
        }
        if !self.filename.is_empty() {
            buf += &format!("({}:{})", self.filename, self.line_number)
                .bright_black()
                .to_string();
        }
        write!(buf, "{} {}", ":".bright_black(), self.message)?;
        for (k, v) in &self.fields {
            buf += &format!(" {k}={v}").bright_black().to_string();
        }

        writer
            .write_fmt(format_args!("{buf}\n"))
            .map_err(Into::into)
    }
}

#[derive(Deserialize, Serialize, Debug)]
struct DetailItem {
    #[serde(flatten)]
    fields: Map<String, Value>,
    #[serde(flatten)]
    span: Map<String, Value>,
}

pub struct LogSender {
    pub tx: Sender<Map<String, Value>>,
}

impl io::Write for LogSender {
    fn write(&mut self, buf: &[u8]) -> io::Result<usize> {
        self.tx
            .send(serde_json::from_slice(buf)?)
            .or(Err(io::ErrorKind::BrokenPipe))?;
        Ok(buf.len())
    }

    fn flush(&mut self) -> io::Result<()> {
        // We do not buffer output.
        Ok(())
    }
}

impl LogSender {
    pub fn new(tx: Sender<Map<String, Value>>) -> impl Fn() -> Self {
        move || Self { tx: tx.clone() }
    }
}

#[derive(Derivative)]
#[derivative(Default(new = "true"))]
pub struct Logger {
    pub verbose: bool,
    pub json: bool,
    pub enable: bool,

    unread: Arc<AtomicU64>,
    tx: Option<Sender<Map<String, Value>>>,
}

impl Logger {
    /// Initialize db connection.
    ///
    /// # Panics
    ///
    /// Panics if the database connection is already set.
    pub fn set_db(db: DatabaseConnection) {
        DB.set(db).unwrap();
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
    #[must_use]
    pub fn get_unread(&self) -> u64 {
        self.unread.load(Ordering::Relaxed)
    }

    /// Write a notification to the database.
    /// When `success` is `true`, `level` is ignored and a special level `Success` is set.
    async fn write_notification(
        unread: Arc<AtomicU64>,
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
            Self::write_pending(db).await;
            let _ = model.insert(db).await;
        } else {
            CACHE.get().unwrap().lock().push(model);
        }
        unread.fetch_add(1, Ordering::SeqCst);
    }

    /// Write pending notifications to database.
    ///
    /// # Panics
    ///
    /// Panics if cache is not initialized.
    pub async fn write_pending(db: &DatabaseConnection) {
        let cached: Vec<_> = CACHE.get().unwrap().lock().drain(..).collect();
        for i in cached {
            let _ = i.insert(db).await;
        }
    }

    /// Init logger, plugins will call this automatically when loaded.
    ///
    /// # Panics
    ///
    /// Panics if logger is not started. Please call [`Self::start_logger`] first.
    pub fn init(&self) {
        if self.enable {
            tracing_subscriber::fmt()
                .with_max_level(if self.verbose {
                    Level::DEBUG
                } else {
                    Level::INFO
                })
                .with_writer(LogSender::new(self.tx.clone().unwrap()))
                .without_time()
                .with_file(true)
                .with_line_number(true)
                .json()
                .init();
        }
    }

    /// Remove useless file path.
    /// Only keep path starting from the last `src`.
    ///
    /// # Examples
    ///
    /// - /root/<u>crate/src/mod/file.rs</u> => crate/src/mod/file.rs
    /// - /root/mypath/src/files/<u>crate/src/main.rs</u> => crate/src/main.rs
    #[must_use]
    pub fn trim_filename(s: &str) -> String {
        let mut buf = Vec::new();
        let mut flag = false;
        let s = PathBuf::from(s);
        for i in s.iter().rev() {
            buf.push(i);
            if flag {
                break;
            }
            if i.eq_ignore_ascii_case("src") {
                flag = true;
            }
        }
        let mut ret = PathBuf::new();
        for i in buf.into_iter().rev() {
            ret.push(i);
        }
        ret.to_string_lossy().into()
    }

    /// Trim target path.
    /// Only keep the first path before `::`. Change several library targets to our own.
    #[must_use]
    pub fn trim_target(s: &str) -> String {
        if s.starts_with("actix_") {
            String::from("skynet-web")
        } else if s.starts_with("sea_orm::") || s.starts_with("sea_orm_migration::") {
            String::from("skynet-db")
        } else {
            s.split("::").next().unwrap_or("unknown").to_owned()
        }
    }

    /// Return colored string of `level`.
    ///
    /// - TRACE/DEBUG => Magenta
    /// - INFO => Green
    /// - WARN => Yellow
    /// - ERROR => Red
    #[must_use]
    pub fn fmt_level(level: &Level) -> String {
        format!("{: >5}", level.to_string())
            .bold()
            .color(match *level {
                Level::TRACE | Level::DEBUG => Color::Magenta,
                Level::INFO => Color::Green,
                Level::WARN => Color::Yellow,
                Level::ERROR => Color::Red,
            })
            .to_string()
    }

    fn is_ignore(target: &str, level: Level) -> bool {
        // ignore lots of https log
        if target.starts_with("h2::") && level > Level::INFO {
            return true;
        }
        // ignore https suite log
        if target.starts_with("rustls::server::hs") && level > Level::INFO {
            return true;
        }
        // ignore middleware log, use our own
        if target.starts_with("tracing_actix_web::middleware")
            || target.starts_with("actix_files::service")
            || target.starts_with("actix_http::h1")
        {
            return true;
        }
        false
    }

    /// Init and start logger.
    ///
    /// # Warning
    ///
    /// Do NOT call this function unless you are sure you know what you are doing.
    ///
    /// # Panics
    ///
    /// Panics if cache is already set.
    pub fn start_logger(&mut self, enable: bool, json: bool, verbose: bool) {
        CACHE.set(Mutex::new(Vec::new())).unwrap();
        self.json = json;
        self.verbose = verbose;
        self.enable = enable;

        if enable {
            let (tx, rx) = mpsc::channel();
            let unread = self.unread.clone();
            self.tx = Some(tx);
            self.init();

            thread::spawn(move || {
                while let Ok(v) = rx.recv() {
                    let success = v
                        .get("success")
                        .and_then(Value::as_bool)
                        .unwrap_or_default();
                    let mut item = LogItem::from_json(v);
                    let level = Level::from_str(&item.level).unwrap_or(Level::ERROR);

                    if Self::is_ignore(&item.target, level) {
                        continue;
                    }

                    item.target = Self::trim_target(&item.target);
                    item.filename = Self::trim_filename(&item.filename);
                    let time = item.fields.remove("_time").unwrap_or_default().as_i64();
                    if !verbose {
                        item.filename.clear();
                        item.line_number = 0;
                    }
                    if level <= Level::WARN || success {
                        // prevent deadlock for sqlite
                        let unread = unread.clone();
                        let target = item.target.clone();
                        let message = item.message.clone();
                        let detail = DetailItem {
                            fields: item.fields.clone(),
                            span: item.span.clone(),
                        };
                        thread::spawn(move || {
                            runtime::Builder::new_current_thread()
                                .enable_time()
                                .build()
                                .unwrap()
                                .block_on(async {
                                    Self::write_notification(
                                        unread,
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

                    let writer: Box<dyn io::Write> = if level <= Level::WARN {
                        Box::new(stderr())
                    } else {
                        Box::new(stdout())
                    };
                    if json {
                        item.time = time.unwrap_or_else(|| Utc::now().timestamp_micros()).into();
                        let _ = item.write_json(writer);
                    } else {
                        item.time = time
                            .map_or_else(Local::now, |v| {
                                DateTime::from_timestamp_micros(v)
                                    .unwrap_or_default()
                                    .into()
                            })
                            .format("%F %T%.6f")
                            .to_string()
                            .into();
                        item.level = Self::fmt_level(&level);
                        let _ = item.write_console(writer);
                    }
                }
            });
        }
    }
}
