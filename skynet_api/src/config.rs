use std::path::Path;

use actix_cloud::config;
use serde::{Deserialize, Serialize};
use serde_inline_default::serde_inline_default;
use validator::{Validate, ValidationError};

fn check_file(path: &str) -> Result<(), ValidationError> {
    if Path::new(path).exists() {
        Ok(())
    } else {
        Err(ValidationError::new("file not exist"))
    }
}

#[serde_inline_default]
#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct ConfigListen {
    #[serde_inline_default("0.0.0.0:8080".into())]
    #[validate(length(min = 1))]
    pub address: String,
    #[serde(default)]
    pub worker: usize,
    #[serde(default)]
    pub ssl: bool,
    #[validate(length(min = 1))]
    pub ssl_cert: Option<String>,
    #[validate(length(min = 1))]
    pub ssl_key: Option<String>,
}

#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct ConfigDatabase {
    #[validate(length(min = 1))]
    pub dsn: String,
}

#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct ConfigRedis {
    #[serde(default)]
    pub enable: bool,
    #[validate(length(min = 1))]
    pub dsn: Option<String>,
}

#[serde_inline_default]
#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct ConfigSession {
    #[validate(length(min = 64))]
    pub key: String,
    #[serde_inline_default("session_".into())]
    #[validate(length(min = 1))]
    pub prefix: String,
    #[serde_inline_default("SESSIONID".into())]
    #[validate(length(min = 1))]
    pub cookie: String,
    #[serde_inline_default(3600)]
    #[validate(range(min = 1))]
    pub expire: u32,
    #[serde_inline_default(5184000)]
    #[validate(range(min = 1))]
    pub remember: u32,
}

#[serde_inline_default]
#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct ConfigHeader {
    #[serde_inline_default("default-src 'none'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; base-uri 'self'".into())]
    #[validate(length(min = 1))]
    pub csp: String,
}

#[serde_inline_default]
#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct ConfigProxy {
    #[serde(default)]
    pub enable: bool,
    #[serde_inline_default("X-Real-Address".into())]
    #[validate(length(min = 1))]
    pub header: String,
}

#[serde_inline_default]
#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct ConfigCsrf {
    #[serde_inline_default("csrf_".into())]
    #[validate(length(min = 1))]
    pub prefix: String,
    #[serde_inline_default(10)]
    #[validate(range(min = 1))]
    pub expire: u32,
}

#[serde_inline_default]
#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct ConfigRecaptcha {
    #[serde(default)]
    pub enable: bool,
    #[serde_inline_default("https://www.recaptcha.net".into())]
    #[validate(length(min = 1))]
    pub url: String,
    #[validate(length(min = 1))]
    pub sitekey: Option<String>,
    #[validate(length(min = 1))]
    pub secret: Option<String>,
    #[serde_inline_default(10)]
    pub timeout: u32,
}

#[serde_inline_default]
#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct ConfigGeoip {
    #[serde(default)]
    pub enable: bool,
    #[serde_inline_default("GeoLite2-City.mmdb".into())]
    pub database: String,
}

#[serde_inline_default]
#[derive(Serialize, Deserialize, Debug, Validate, Clone)]
pub struct Config {
    #[validate(nested)]
    pub database: ConfigDatabase,
    #[validate(nested)]
    pub redis: ConfigRedis,
    #[validate(nested)]
    pub session: ConfigSession,
    #[validate(nested)]
    pub listen: ConfigListen,
    #[validate(nested)]
    pub header: ConfigHeader,
    #[validate(nested)]
    pub proxy: ConfigProxy,
    #[validate(nested)]
    pub recaptcha: ConfigRecaptcha,
    #[validate(nested)]
    pub csrf: ConfigCsrf,
    #[validate(nested)]
    pub geoip: ConfigGeoip,
    #[serde_inline_default("default.webp".into())]
    #[validate(custom(function = "check_file"))]
    pub avatar: String,
    #[serde_inline_default("en-US".into())]
    #[validate(length(min = 1))]
    pub lang: String,
}

pub fn load_file<S: AsRef<str>>(path: S) -> (config::Config, Config) {
    let cfg = config::Config::builder()
        .add_source(config::File::with_name(path.as_ref()))
        .build()
        .expect("Config load failed");
    let ret: Config = cfg.clone().try_deserialize().expect("Config file invalid");
    let _ = ret.validate().map_err(|e| panic!("{}", e.to_string()));
    // additional checker
    if ret.listen.ssl {
        assert!(
            ret.listen.ssl_key.is_some(),
            "`listen.ssl_key` is not provided"
        );
        assert!(
            ret.listen.ssl_cert.is_some(),
            "`listen.ssl_cert` is not provided"
        );
    }
    if ret.recaptcha.enable {
        assert!(
            ret.recaptcha.sitekey.is_some(),
            "`recaptcha.sitekey` is not provided"
        );
        assert!(
            ret.recaptcha.secret.is_some(),
            "`recaptcha.secret` is not provided"
        );
    }
    if ret.redis.enable {
        assert!(ret.redis.dsn.is_some(), "`redis.dsn` is not provided");
    }
    if ret.geoip.enable {
        assert!(
            check_file(&ret.geoip.database).is_ok(),
            "`geoip.database` file not exist"
        );
    }
    (cfg, ret)
}
