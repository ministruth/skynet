use std::{any::type_name, path::Path};

use derivative::Derivative;
use rs_config::ConfigError;
use skynet_macro::{Foreach, Iterable};
use tracing::warn;

macro_rules! checker_file {
    () => {
        Some(Box::new(|n, v, _| {
            assert!(Path::new(v).exists(), "`{n}` file not found");
        }))
    };
}

macro_rules! checker_str_need_when {
    ($n:ident) => {
        Some(Box::new(|n, v, c| {
            if c.$n.get() {
                assert!(!v.is_empty(), "`{n}` is not provided");
            }
        }))
    };
}

macro_rules! checker_ge {
    ($n:literal) => {
        Some(Box::new(|n, v, _| {
            assert!(*v >= $n, "`{n}` needs to be greater or equal to {}", $n);
        }))
    };
}

#[allow(unused_macros)]
macro_rules! checker_range {
    ($n1:literal, $n2:literal) => {
        Some(Box::new(|n, v, _| {
            assert!(*v >= $n1, "`{n}` needs to be greater or equal to {}", $n1);
            assert!(*v <= $n2, "`{n}` needs to be less or equal to {}", $n2);
        }))
    };
}

macro_rules! checker_str_ge {
    ($n:literal) => {
        Some(Box::new(|n, v, _| {
            assert!(v.len() >= $n, "`{n}` needs at least {} in length", $n);
        }))
    };
}

#[derive(Derivative, Foreach, Iterable)]
#[derivative(Debug)]
pub struct Config {
    pub database_dsn: ConfigItem<String>,

    pub redis_dsn: ConfigItem<String>,

    pub session_key: ConfigItem<String>,
    pub session_prefix: ConfigItem<String>,
    pub session_cookie: ConfigItem<String>,
    pub session_expire: ConfigItem<i64>,
    pub session_remember: ConfigItem<i64>,

    pub listen_address: ConfigItem<String>,
    pub listen_worker: ConfigItem<i64>,
    pub listen_ssl: ConfigItem<bool>,
    pub listen_ssl_cert: ConfigItem<String>,
    pub listen_ssl_key: ConfigItem<String>,

    pub header_csp: ConfigItem<String>,

    pub proxy_enable: ConfigItem<bool>,
    pub proxy_header: ConfigItem<String>,
    pub proxy_trusted: ConfigItem<String>,

    pub recaptcha_enable: ConfigItem<bool>,
    pub recaptcha_url: ConfigItem<String>,
    pub recaptcha_sitekey: ConfigItem<String>,
    pub recaptcha_secret: ConfigItem<String>,
    pub recaptcha_timeout: ConfigItem<i64>,

    pub csrf_prefix: ConfigItem<String>,
    pub csrf_timeout: ConfigItem<i64>,

    pub avatar: ConfigItem<String>,
    pub lang: ConfigItem<String>,
}

impl Default for Config {
    fn default() -> Self {
        Self::new()
    }
}

impl Config {
    #[must_use]
    #[allow(clippy::too_many_lines, clippy::missing_panics_doc)]
    pub fn new() -> Self {
        Self {
            database_dsn: ConfigItem {
                name: "database.dsn".to_owned(),
                default: String::new(),
                required: true,
                ..Default::default()
            },

            redis_dsn: ConfigItem {
                name: "redis.dsn".to_owned(),
                default: String::new(),
                required: true,
                ..Default::default()
            },

            session_key: ConfigItem {
                name: "session.key".to_owned(),
                default: "Ckm7cYj0XDaIkGQeXYun7fuduCT8V5dMwYzMAz2mb5nlj3FTAgLYp5MstHh18PW8"
                    .to_owned(),
                warning: true,
                checker: checker_str_ge!(64),
                ..Default::default()
            },
            session_prefix: ConfigItem {
                name: "session.prefix".to_owned(),
                default: "session_".to_owned(),
                checker: checker_str_ge!(1),
                ..Default::default()
            },
            session_cookie: ConfigItem {
                name: "session.cookie".to_owned(),
                default: "SESSIONID".to_owned(),
                checker: checker_str_ge!(1),
                ..Default::default()
            },
            session_expire: ConfigItem {
                name: "session.expire".to_owned(),
                default: 3600,
                checker: checker_ge!(1),
                ..Default::default()
            },
            session_remember: ConfigItem {
                name: "session.remember".to_owned(),
                default: 5_184_000,
                checker: checker_ge!(1),
                ..Default::default()
            },

            listen_address: ConfigItem {
                name: "listen.address".to_owned(),
                default: "0.0.0.0:8080".to_owned(),
                checker: checker_str_ge!(1),
                ..Default::default()
            },
            listen_worker: ConfigItem {
                name: "listen.worker".to_owned(),
                default: 0,
                checker: checker_ge!(0),
                ..Default::default()
            },
            listen_ssl: ConfigItem {
                name: "listen.ssl".to_owned(),
                default: false,
                warning:true,
                ..Default::default()
            },
            listen_ssl_cert: ConfigItem {
                name: "listen.ssl_cert".to_owned(),
                default: String::new(),
                checker: Some(Box::new(|n, v, c| {
                    if c.listen_ssl.get() {
                        assert!(!v.is_empty(), "`{n}` is not provided");
                        assert!(Path::new(v).exists(), "`{n}` file not found");
                    }
                })),
                ..Default::default()
            },
            listen_ssl_key: ConfigItem {
                name: "listen.ssl_key".to_owned(),
                default: String::new(),
                checker: Some(Box::new(|n, v, c| {
                    if c.listen_ssl.get() {
                        assert!(!v.is_empty(), "`{n}` is not provided");
                        assert!(Path::new(v).exists(), "`{n}` file not found");
                    }
                })),
                ..Default::default()
            },

            header_csp: ConfigItem {
                name: "header.csp".to_owned(),
                default: "default-src 'none'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; base-uri 'self'".to_owned(),
                checker: checker_str_ge!(1),
                ..Default::default()
            },

            proxy_enable: ConfigItem {
                name: "proxy.enable".to_owned(),
                default: false,
                ..Default::default()
            },
            proxy_header: ConfigItem {
                name: "proxy.header".to_owned(),
                default: "X-Forwarded-For".to_owned(),
                checker: checker_str_need_when!(proxy_enable),
                ..Default::default()
            },
            proxy_trusted: ConfigItem {
                name: "proxy.trusted".to_owned(),
                default: String::new(),
                checker: checker_str_need_when!(proxy_enable),
                ..Default::default()
            },

            recaptcha_enable: ConfigItem {
                name: "recaptcha.enable".to_owned(),
                default: false,
                warning: true,
                public: true,
                ..Default::default()
            },
            recaptcha_url: ConfigItem {
                name: "recaptcha.url".to_owned(),
                default: "https://www.recaptcha.net" .to_owned(),
                checker: checker_str_need_when!(recaptcha_enable),
                public: true,
                ..Default::default()
            },
            recaptcha_sitekey: ConfigItem {
                name: "recaptcha.sitekey".to_owned(),
                default: String::new(),
                checker: checker_str_need_when!(recaptcha_enable),
                public: true,
                ..Default::default()
            },
            recaptcha_secret: ConfigItem {
                name: "recaptcha.secret".to_owned(),
                default: String::new(),
                checker: checker_str_need_when!(recaptcha_enable),
                ..Default::default()
            },
            recaptcha_timeout: ConfigItem {
                name: "recaptcha.timeout".to_owned(),
                default: 10,
                checker: checker_ge!(0),
                ..Default::default()
            },

            csrf_prefix: ConfigItem {
                name: "csrf.prefix".to_owned(),
                default: "csrf_".to_owned(),
                checker: checker_str_ge!(1),
                ..Default::default()
            },
            csrf_timeout: ConfigItem {
                name: "csrf.timeout".to_owned(),
                default: 10,
                checker: checker_ge!(1),
                ..Default::default()
            },

            avatar: ConfigItem {
                name: "avatar".to_owned(),
                default: "default.webp".to_owned(),
                checker: checker_file!(),
                ..Default::default()
            },
            lang: ConfigItem {
                name: "lang".to_owned(),
                default: "en-US".to_owned(),
                checker: checker_str_ge!(1),
                public: true,
                ..Default::default()
            },
        }
    }
}

pub trait ItemType {}

impl ItemType for String {}
impl ItemType for i64 {}
impl ItemType for f64 {}
impl ItemType for bool {}

type ConfigChecker<T> = Box<dyn Fn(&str, &T, &Config) + Send + Sync>;

#[derive(Derivative)]
#[derivative(Debug, Default)]
pub struct ConfigItem<T>
where
    T: ItemType + Clone + PartialEq,
{
    /// Config name.
    pub name: String,
    /// Config default.
    pub default: T,
    /// Config value.
    value: T,
    /// Whether public to guest user.
    pub public: bool,
    /// User must give value.
    required: bool,
    /// Whether show warning for default value.
    warning: bool,
    /// Config checker.
    #[derivative(Debug = "ignore")]
    checker: Option<ConfigChecker<T>>,
}

impl ConfigItem<i64> {
    #[must_use]
    #[allow(clippy::missing_const_for_fn)]
    pub fn get(&self) -> i64 {
        self.value
    }
}

impl ConfigItem<f64> {
    #[must_use]
    #[allow(clippy::missing_const_for_fn)]
    pub fn get(&self) -> f64 {
        self.value
    }
}

impl ConfigItem<bool> {
    #[must_use]
    #[allow(clippy::missing_const_for_fn)]
    pub fn get(&self) -> bool {
        self.value
    }
}

impl ConfigItem<String> {
    #[must_use]
    #[allow(clippy::missing_const_for_fn)]
    pub fn get(&self) -> &str {
        &self.value
    }
}

impl<T> ConfigItem<T>
where
    T: ItemType + Clone + PartialEq,
{
    /// # Panics
    ///
    /// Panics if the config item is invalid, with a message showing the error.
    pub fn parse(&mut self, value: Result<T, ConfigError>) {
        self.value = value.unwrap_or_else(|e| match e {
            ConfigError::NotFound(_) => self.default.clone(),
            _ => panic!("Config `{}` needs to be {}", self.name, type_name::<T>()),
        });
        if self.value == self.default {
            assert!(!self.required, "Config `{}` must be given", self.name);
            if self.warning {
                warn!(
                    "Config `{}` has default value, please modify your config file for safety",
                    self.name
                );
            }
        }
    }
}

/// # Panics
///
/// Panics if the config file is invalid.
#[allow(clippy::cognitive_complexity)]
pub fn load_file<S: AsRef<str>>(path: S) -> Config {
    let mut ret = Config::default();
    let c = rs_config::Config::builder()
        .add_source(rs_config::File::with_name(path.as_ref()))
        .build()
        .expect("Config load failed");

    for (_, v) in ret.iter_mut() {
        if let Some(v) = v.downcast_mut::<ConfigItem<String>>() {
            v.parse(c.get_string(&v.name));
        } else if let Some(v) = v.downcast_mut::<ConfigItem<i64>>() {
            v.parse(c.get_int(&v.name));
        } else if let Some(v) = v.downcast_mut::<ConfigItem<bool>>() {
            v.parse(c.get_bool(&v.name));
        } else if let Some(v) = v.downcast_mut::<ConfigItem<f64>>() {
            v.parse(c.get_float(&v.name));
        } else {
            unreachable!()
        }
    }
    foreach_Config!(
        ret,
        v,
        if let Some(x) = &v.checker {
            x(&v.name, &v.value, &ret);
        }
    );
    ret
}
