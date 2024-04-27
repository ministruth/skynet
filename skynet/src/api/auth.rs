use std::{
    collections::HashMap,
    time::{self},
};

use actix_session::Session;
use actix_web::{cookie::time::Duration, web::Data, Responder};
use actix_web_validator::Json;
use anyhow::{anyhow, bail, Result};
use awc::Client;
use log::warn;
use redis::aio::ConnectionManager;
use sea_orm::{DatabaseConnection, TransactionTrait};
use serde::{Deserialize, Serialize};
use serde_json::json;
use skynet::{
    finish,
    request::{Request, Response, ResponseCode, ResponseCookie, RspResult},
    success, HyUuid, Skynet,
};
use validator::Validate;

use crate::api::{new_csrf_token, APIError, CSRF_COOKIE};

#[derive(Debug, Validate, Deserialize)]
pub struct SigninReq {
    #[validate(length(max = 32))]
    username: String,
    password: String,
    remember: Option<bool>,
    #[serde(rename = "g-recaptcha-response")]
    recaptcha: Option<String>,
}

pub async fn signin(
    param: Json<SigninReq>,
    db: Data<DatabaseConnection>,
    req: Request,
    session: Session,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    if skynet.config.recaptcha_enable.get() {
        if let Some(x) = &param.recaptcha {
            let timeout = skynet.config.recaptcha_timeout.get();
            if verify_recaptcha(
                x.to_owned(),
                req.ip.ip().to_string(),
                RecaptchaOption {
                    url: skynet.config.recaptcha_url.get().to_owned(),
                    secret: skynet.config.recaptcha_secret.get().to_owned(),
                    timeout: if timeout == 0 {
                        None
                    } else {
                        Some(time::Duration::from_secs(timeout.try_into()?))
                    },
                },
            )
            .await
            .is_err()
            {
                finish!(Response::new(ResponseCode::CodeRecaptchaInvalid));
            }
        } else {
            finish!(Response::bad_request(
                APIError::MissingField("recaptcha".to_owned()).to_string(),
            ));
        }
    }

    let tx = db.begin().await?;
    let (ok, user) = skynet
        .user
        .check_pass(&tx, &param.username, &param.password)
        .await?;
    if !ok {
        warn!(
            "Invalid username or password, attempt user: {}",
            param.username
        );
        finish!(Response::new(ResponseCode::CodeUserInvalid));
    }
    let user = skynet
        .user
        .update_login(&tx, &user.unwrap().id, &req.ip.ip().to_string())
        .await?;
    tx.commit().await?;

    let log_detail = json!({
        "id":&user.id,
        "name":&user.username,
        "ip":&user.last_ip,
    });
    session.renew();
    session.insert("id", user.id)?;
    session.insert("name", user.username.clone())?;
    session.insert("time", user.last_login.unwrap())?;
    if param.remember.is_some_and(|x| x) {
        session.insert("_ttl", skynet.config.session_remember.get())?;
    } else {
        session.insert("_ttl", skynet.config.session_expire.get())?;
    }
    success!("User signin\n{}", log_detail.to_string());
    finish!(Response::ok());
}

#[derive(Debug)]
struct RecaptchaOption {
    url: String,
    secret: String,
    timeout: Option<time::Duration>,
}

async fn verify_recaptcha(response: String, ip: String, option: RecaptchaOption) -> Result<()> {
    #[derive(Deserialize, Serialize)]
    struct Response {
        success: bool,
        challenge_ts: String,
        hostname: String,
        #[serde(rename = "error-codes")]
        error_codes: Vec<String>,
    }
    let client = Client::default();
    let mut req = client.post(option.url + "/recaptcha/api/siteverify");
    if let Some(x) = option.timeout {
        req = req.timeout(x);
    }
    let mut rsp = req
        .send_json(&json!({
            "secret": option.secret,
            "remoteip": ip,
            "response": response,
        }))
        .await
        .map_err(|e| anyhow!(e.to_string()))?;
    let rsp = rsp.json::<Response>().await?;
    if !rsp.error_codes.is_empty() {
        bail!("remote error codes: {:?}", rsp.error_codes)
    }
    if !rsp.success {
        bail!("invalid challenge solution or remote IP")
    }
    Ok(())
}

pub async fn signout(session: Session) -> RspResult<impl Responder> {
    session.purge();
    finish!(Response::ok());
}

pub async fn get_access(req: Request) -> RspResult<impl Responder> {
    #[derive(Serialize)]
    struct Rsp {
        signin: bool,
        #[serde(skip_serializing_if = "Option::is_none")]
        id: Option<HyUuid>,
        permission: HashMap<String, i32>,
    }
    let mut rsp = Rsp {
        signin: false,
        id: None,
        permission: HashMap::new(),
    };
    if let Some(id) = req.uid {
        rsp.signin = true;
        rsp.id = Some(id);
    }
    req.perm.into_iter().for_each(|(_, v)| {
        rsp.permission.insert(v.name, v.perm);
    });
    finish!(Response::data(rsp));
}

pub async fn get_token(
    skynet: Data<Skynet>,
    redis: Data<ConnectionManager>,
) -> RspResult<impl Responder> {
    let token = new_csrf_token(&skynet, &redis).await?;
    finish!(Response::ok().add_cookie(ResponseCookie {
        name: CSRF_COOKIE.to_owned(),
        value: token,
        max_age: Duration::seconds(skynet.config.csrf_timeout.get()),
        http_only: false, // http_only must not be set
        secure: skynet.config.listen_ssl.get(),
        ..Default::default()
    }));
}
