use actix_cloud::{
    actix_web::web::Data,
    response::{JsonResponse, RspResult},
    utils,
};
use actix_web_validator::Json;
use base64::{prelude::BASE64_URL_SAFE_NO_PAD, Engine};
use openssl::{
    bn::BigNumContext,
    ec::{self},
    nid,
};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use skynet_api::{
    config::{
        CONFIG_SESSION_EXPIRE, CONFIG_SESSION_KEY, CONFIG_SESSION_REMEMBER,
        CONFIG_WEBPUSH_ENDPOINT, CONFIG_WEBPUSH_KEY,
    },
    request::unique_validator,
    sea_orm::{DatabaseConnection, TransactionTrait},
    tracing::info,
    viewer::{settings::SettingViewer, webpush_clients::WebpushClientViewer},
    Skynet,
};
use validator::Validate;

use crate::{finish_data, finish_ok, request::APIError, webpush::WebpushManager};

pub async fn get_public(
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<JsonResponse> {
    #[derive(Serialize)]
    struct Rsp {
        #[serde(rename = "recaptcha.enable")]
        recaptcha_enable: bool,
        #[serde(rename = "recaptcha.url")]
        recaptcha_url: String,
        #[serde(rename = "recaptcha.sitekey", skip_serializing_if = "Option::is_none")]
        recaptcha_sitekey: Option<String>,
        lang: String,
        #[serde(rename = "webpush.key")]
        webpush_key: String,
        #[serde(rename = "geoip.enable")]
        geoip_enable: bool,
    }
    // webpush_key
    let key = SettingViewer::get_base64(db.as_ref(), CONFIG_WEBPUSH_KEY)
        .await?
        .ok_or(APIError::MissingSetting(CONFIG_WEBPUSH_KEY.to_owned()))?;
    let key = ec::EcKey::private_key_from_pem(&key)?;
    let mut ctx = BigNumContext::new()?;
    let group = ec::EcGroup::from_curve_name(nid::Nid::X9_62_PRIME256V1)?;
    let keybytes =
        key.public_key()
            .to_bytes(&group, ec::PointConversionForm::UNCOMPRESSED, &mut ctx)?;
    let webpush_key = BASE64_URL_SAFE_NO_PAD.encode(&keybytes);

    let ret = Rsp {
        recaptcha_enable: skynet.config.recaptcha.enable,
        recaptcha_url: skynet.config.recaptcha.url.clone(),
        recaptcha_sitekey: skynet.config.recaptcha.sitekey.clone(),
        lang: skynet.config.lang.clone(),
        webpush_key,
        geoip_enable: skynet.config.geoip.enable,
    };
    finish_data!(ret);
}

pub async fn get_system(db: Data<DatabaseConnection>) -> RspResult<JsonResponse> {
    #[derive(Serialize)]
    struct Rsp {
        #[serde(rename = "session.expire")]
        session_expire: u32,
        #[serde(rename = "session.remember")]
        session_remember: u32,
        #[serde(rename = "webpush.endpoint")]
        webpush_endpoint: Vec<String>,
    }
    let session_expire = SettingViewer::get(db.as_ref(), CONFIG_SESSION_EXPIRE)
        .await?
        .ok_or(APIError::MissingSetting(CONFIG_SESSION_EXPIRE.to_owned()))?
        .parse::<u32>()?;
    let session_remember = SettingViewer::get(db.as_ref(), CONFIG_SESSION_REMEMBER)
        .await?
        .ok_or(APIError::MissingSetting(CONFIG_SESSION_REMEMBER.to_owned()))?
        .parse::<u32>()?;
    let webpush_endpoint = SettingViewer::get(db.as_ref(), CONFIG_WEBPUSH_ENDPOINT)
        .await?
        .ok_or(APIError::MissingSetting(CONFIG_WEBPUSH_ENDPOINT.to_owned()))?;
    let webpush_endpoint = serde_json::from_str::<Value>(&webpush_endpoint)?
        .as_array()
        .ok_or(APIError::InvalidSetting(CONFIG_WEBPUSH_ENDPOINT.to_owned()))?
        .iter()
        .map(|x| x.as_str().unwrap_or("").to_owned())
        .filter(|x| !x.is_empty())
        .collect();

    finish_data!(Rsp {
        session_expire,
        session_remember,
        webpush_endpoint,
    });
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutSystemReq {
    #[serde(rename = "session.expire")]
    pub session_expire: Option<u32>,
    #[serde(rename = "session.remember")]
    pub session_remember: Option<u32>,
    #[validate(custom(function = "unique_validator"))]
    #[serde(rename = "webpush.endpoint")]
    webpush_endpoint: Option<Vec<String>>,
}

pub async fn put_system(
    param: Json<PutSystemReq>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    let tx = db.begin().await?;
    if let Some(x) = param.session_expire {
        SettingViewer::set(&tx, CONFIG_SESSION_EXPIRE, &x.to_string()).await?;
    }
    if let Some(x) = param.session_remember {
        SettingViewer::set(&tx, CONFIG_SESSION_REMEMBER, &x.to_string()).await?;
    }
    if let Some(x) = &param.webpush_endpoint {
        SettingViewer::set(&tx, CONFIG_WEBPUSH_ENDPOINT, &serde_json::to_string(x)?).await?;
    }
    tx.commit().await?;

    info!(success = true, "Put system");
    finish_ok!();
}

pub async fn reset_webpush_key(
    db: Data<DatabaseConnection>,
    webpush: Data<WebpushManager>,
) -> RspResult<JsonResponse> {
    let group = ec::EcGroup::from_curve_name(nid::Nid::X9_62_PRIME256V1)?;
    let key = ec::EcKey::generate(&group)?;
    let key = key.private_key_to_pem()?;
    let tx = db.begin().await?;
    SettingViewer::set_base64(&tx, CONFIG_WEBPUSH_KEY, &key).await?;
    WebpushClientViewer::delete_all(&tx).await?;
    tx.commit().await?;
    *webpush.key.write() = key;
    info!(success = true, "Reset webpush key");
    finish_ok!();
}

pub async fn reset_session_key(
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<JsonResponse> {
    let key = utils::rand_string(64);
    let tx = db.begin().await?;
    SettingViewer::set(&tx, CONFIG_SESSION_KEY, &key).await?;
    tx.commit().await?;
    skynet.warning.insert(
        String::from("session"),
        String::from("text.warning.session"),
    );
    info!(success = true, "Reset session key");
    finish_ok!();
}
