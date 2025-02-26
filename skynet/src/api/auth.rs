use std::collections::HashMap;

use actix_cloud::{
    actix_web::{
        HttpRequest,
        cookie::{Cookie, SameSite, time::Duration},
        http::header::USER_AGENT,
        web::Data,
    },
    response::{JsonResponse, RspResult},
    session::Session,
    state::GlobalState,
    tracing::{info, warn},
};
use actix_web_validator::Json;
use serde::{Deserialize, Serialize};
use skynet_api::{
    HyUuid, Skynet,
    config::{CONFIG_SESSION_EXPIRE, CONFIG_SESSION_REMEMBER},
    finish,
    request::{self, Request},
    sea_orm::{DatabaseConnection, TransactionTrait},
    viewer::{settings::SettingViewer, users::UserViewer},
};
use validator::Validate;

use crate::{
    SkynetResponse,
    api::client::{RecaptchaOption, verify_recaptcha},
    finish_data, finish_err, finish_ok,
    request::{APIError, CSRF_COOKIE, new_csrf_token},
};

#[derive(Debug, Validate, Deserialize)]
pub struct SigninReq {
    #[validate(length(max = 32))]
    pub username: String,
    pub password: String,
    pub remember: Option<bool>,
    #[serde(rename = "g-recaptcha-response")]
    pub recaptcha: Option<String>,
}

pub async fn signin(
    r: HttpRequest,
    req: Request,
    param: Json<SigninReq>,
    skynet: Data<Skynet>,
    session: Session,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    if skynet.config.recaptcha.enable {
        if let Some(x) = &param.recaptcha {
            if verify_recaptcha(
                skynet.clone(),
                x.to_owned(),
                req.extension.real_ip.ip().to_string(),
                RecaptchaOption {
                    url: skynet.config.recaptcha.url.clone(),
                    secret: skynet.config.recaptcha.secret.clone().unwrap(),
                },
            )
            .await
            .is_err()
            {
                finish_err!(SkynetResponse::RecaptchaInvalid);
            }
        } else {
            finish!(JsonResponse::bad_request(
                APIError::MissingField(String::from("recaptcha")).to_string()
            ));
        }
    }

    let tx = db.begin().await?;
    let (ok, user) = UserViewer::check_pass(&tx, &param.username, &param.password).await?;
    if !ok {
        warn!(username = param.username, "Invalid username or password");
        finish_err!(SkynetResponse::UserInvalid);
    }
    let user_agent: Option<String> = r
        .headers()
        .get(USER_AGENT)
        .and_then(|x| x.to_str().ok())
        .map(|x| x.chars().take(512).collect());
    let user = UserViewer::update_login(
        &tx,
        &user.unwrap().id,
        &req.extension.real_ip.ip().to_string(),
        user_agent.as_deref(),
    )
    .await?;
    let ttl = if param.remember.is_some_and(|x| x) {
        SettingViewer::get(&tx, CONFIG_SESSION_REMEMBER)
            .await?
            .ok_or(APIError::MissingSetting(CONFIG_SESSION_REMEMBER.to_owned()))?
            .parse::<u32>()?
    } else {
        SettingViewer::get(&tx, CONFIG_SESSION_EXPIRE)
            .await?
            .ok_or(APIError::MissingSetting(CONFIG_SESSION_EXPIRE.to_owned()))?
            .parse::<u32>()?
    };
    tx.commit().await?;

    session.renew();
    request::Session {
        id: user.id,
        name: user.username.clone(),
        ttl,
        time: user.last_login.unwrap(),
        user_agent,
    }
    .into_session(session)?;
    info!(success = true, id = %user.id, name = user.username, "User signin");
    finish_ok!();
}

pub async fn signout(session: Session) -> RspResult<JsonResponse> {
    session.purge();
    finish_ok!();
}

pub async fn get_access(req: Request) -> RspResult<JsonResponse> {
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
    finish_data!(rsp);
}

pub async fn get_token(state: Data<GlobalState>, skynet: Data<Skynet>) -> RspResult<JsonResponse> {
    let token = new_csrf_token(&skynet, &state).await?;
    finish!(
        JsonResponse::new(SkynetResponse::Success).builder(move |r| {
            r.cookie(
                Cookie::build(CSRF_COOKIE, &token)
                    .max_age(Duration::seconds(skynet.config.csrf.expire.into()))
                    .http_only(false)
                    .path("/")
                    .same_site(SameSite::Strict)
                    .secure(skynet.config.listen.ssl)
                    .finish(),
            );
        })
    );
}
