use std::{
    collections::HashMap,
    fmt::{self, Display},
    net::SocketAddr,
    pin::Pin,
};

use actix_http::{body::EitherBody, HttpMessage, Payload, StatusCode};
use actix_session::Session;
use actix_web::{
    cookie::{time::Duration, Cookie, SameSite},
    http::header::ContentType,
    web::Data,
    FromRequest, HttpRequest, HttpResponse, Responder, ResponseError, Route,
};
use chrono::{DateTime, Utc};
use derivative::Derivative;
use enum_as_inner::EnumAsInner;
use futures::Future;
use qstring::QString;
use sea_orm::{DatabaseConnection, Order, TransactionTrait};
use serde::{Deserialize, Serialize};
use serde_inline_default::serde_inline_default;
use serde_json::json;
use serde_with::{serde_as, DisplayFromStr};
use std::hash::Hash;
use validator::{Validate, ValidationError};

use crate::{
    permission::{PermEntry, PermissionItem, GUEST_ID, PERM_ALL, ROOT_ID, USER_ID},
    t, utils, HyUuid, Skynet,
};

#[macro_export]
macro_rules! finish {
    ($rsp:expr) => {
        return Ok($rsp)
    };
}

/// # Errors
/// Will return `Err` when `x` has duplicate items.
pub fn unique_validator<T: Eq + Hash>(x: &Vec<T>) -> Result<(), ValidationError> {
    if utils::is_unique(x) {
        Ok(())
    } else {
        Err(ValidationError::new("not unique"))
    }
}

#[macro_export]
macro_rules! like_expr {
    ($col:expr, $txt:expr) => {
        skynet::sea_orm::prelude::Expr::col(($col.entity_name(), $col)).like(
            skynet::sea_orm::sea_query::LikeExpr::new(format!(
                "%{}%",
                skynet::utils::like_escape($txt)
            ))
            .escape('\\'),
        )
    };
}

#[derive(Debug, Validate, Deserialize)]
pub struct IDsReq {
    #[validate(length(min = 1, max = 32), custom = "unique_validator")]
    pub id: Vec<HyUuid>,
}

#[serde_as]
#[serde_inline_default]
#[derive(Derivative, Serialize, Deserialize, Validate, Clone)]
#[derivative(Debug, Default(new = "true"))]
pub struct PaginationParam {
    #[validate(range(min = 1))]
    #[serde_inline_default(1)]
    #[derivative(Default(value = "1"))]
    #[serde_as(as = "DisplayFromStr")]
    pub page: u64,

    #[validate(range(min = 1))]
    #[serde_inline_default(10)]
    #[derivative(Default(value = "10"))]
    #[serde_as(as = "DisplayFromStr")]
    pub size: u64,
}

impl PaginationParam {
    #[must_use]
    pub const fn left(&self) -> u64 {
        (self.page - 1) * self.size
    }

    #[must_use]
    pub const fn right(&self) -> u64 {
        self.page * self.size
    }

    /// # Panics
    /// Panics when size too long.
    #[must_use]
    pub fn split<T>(&self, data: Vec<T>) -> PageData<T> {
        let cnt = data.len() as u64;
        PageData::new((
            data.into_iter()
                .skip(self.left().try_into().unwrap())
                .take(self.size.try_into().unwrap())
                .collect(),
            cnt,
        ))
    }
}

#[derive(Serialize, Derivative)]
#[derivative(Debug)]
pub struct PageData<T> {
    total: u64,
    data: Vec<T>,
}

impl<T> PageData<T> {
    #[must_use]
    pub fn new(data: (Vec<T>, u64)) -> Self {
        Self {
            total: data.1,
            data: data.0,
        }
    }
}

#[derive(Debug, Serialize, Deserialize, Clone, Copy, EnumAsInner)]
pub enum SortType {
    #[serde(rename = "asc")]
    ASC,
    #[serde(rename = "desc")]
    DESC,
}

impl From<SortType> for Order {
    fn from(val: SortType) -> Self {
        match val {
            SortType::ASC => Self::Asc,
            SortType::DESC => Self::Desc,
        }
    }
}

#[serde_as]
#[derive(Debug, Serialize, Deserialize, Validate)]
pub struct TimeParam {
    pub created_sort: Option<SortType>,
    #[validate(range(min = 0))]
    #[serde_as(as = "Option<DisplayFromStr>")]
    pub created_start: Option<i64>,
    #[validate(range(min = 0))]
    #[serde_as(as = "Option<DisplayFromStr>")]
    pub created_end: Option<i64>,

    pub updated_sort: Option<SortType>,
    #[validate(range(min = 0))]
    #[serde_as(as = "Option<DisplayFromStr>")]
    pub updated_start: Option<i64>,
    #[validate(range(min = 0))]
    #[serde_as(as = "Option<DisplayFromStr>")]
    pub updated_end: Option<i64>,
}

#[macro_export]
macro_rules! build_time_cond {
    ($cond:ident, $time:expr, $col:expr) => {{
        $cond = $cond.add_option(
            $time
                .created_start
                .map(|x| paste::paste!($col::CreatedAt.gte(x))),
        );
        $cond = $cond.add_option(
            $time
                .created_end
                .map(|x| paste::paste!($col::CreatedAt.lte(x))),
        );
        $cond = $cond.add_option(
            $time
                .updated_start
                .map(|x| paste::paste!($col::UpdatedAt.gte(x))),
        );
        $cond = $cond.add_option(
            $time
                .updated_end
                .map(|x| paste::paste!($col::UpdatedAt.lte(x))),
        );
        if let Some(x) = $time.created_sort {
            $cond = $cond.add_sort(paste::paste!($col::CreatedAt.into_simple_expr()), x.into())
        };
        if let Some(x) = $time.updated_sort {
            $cond = $cond.add_sort(paste::paste!($col::UpdatedAt.into_simple_expr()), x.into())
        };
        $cond
    }};
}

pub type PermChecker = dyn Fn(&HashMap<HyUuid, PermissionItem>) -> bool;

pub enum PermType {
    Entry(PermEntry),
    Custom(Box<PermChecker>),
}

pub struct APIRoute {
    pub path: String,
    pub route: Route,
    pub permission: PermType,
    pub ws_csrf: bool,
}

impl Default for APIRoute {
    fn default() -> Self {
        Self {
            path: String::new(),
            route: Route::new(),
            permission: PermType::Entry(PermEntry::new_root()), // default pemission is root
            ws_csrf: false,
        }
    }
}

/// Get `req` real ip, handle proxy automatically.
///
/// # Panics
///
/// Panics if ip cannot be parsed.
pub fn get_real_ip(req: &HttpRequest, skynet: &Skynet) -> SocketAddr {
    let mut ip = req.peer_addr().unwrap();
    if skynet.config.proxy_enable.get() {
        let trusted: Vec<&str> = skynet
            .config
            .proxy_trusted
            .get()
            .split(',')
            .map(str::trim)
            .collect();
        for i in req
            .headers()
            .get_all(skynet.config.proxy_header.get())
            .map(|x| x.to_str().unwrap())
            .rev()
        {
            if !trusted.contains(&i) {
                ip = i.parse().unwrap();
                break;
            }
        }
    }
    ip
}

#[derive(Derivative)]
#[derivative(Debug)]
pub struct Request {
    /// user id
    pub uid: Option<HyUuid>,

    /// user name
    pub username: Option<String>,

    /// User permission.
    pub perm: HashMap<HyUuid, PermissionItem>,

    /// Parsed language string, fallback default if not provided.
    pub lang: String,

    /// Real ip address.
    pub ip: SocketAddr,

    pub start_time: DateTime<Utc>,

    pub request_id: HyUuid,
}

#[derive(Derivative)]
#[derivative(Default(new = "true"))]
pub struct RequestExtension {
    pub start_time: DateTime<Utc>,
    pub lang: String,
    pub request_id: HyUuid,
}

impl FromRequest for Request {
    type Error = actix_web::Error;
    type Future = Pin<Box<dyn Future<Output = Result<Self, Self::Error>>>>;

    #[inline]
    fn from_request(req: &HttpRequest, _: &mut Payload) -> Self::Future {
        let s = Session::get_session(&mut req.extensions_mut());
        let req = req.clone();
        Box::pin(async move {
            // return cached to prevent reparse.
            if let Some(x) = req.extensions_mut().remove::<Self>() {
                return Ok(x);
            }
            let skynet = req.app_data::<Data<Skynet>>().unwrap();
            let db = req.app_data::<Data<DatabaseConnection>>().unwrap();
            let query_str = req.query_string();
            let qs = QString::from(query_str);
            let lang = qs
                .get("lang")
                .unwrap_or_else(|| skynet.config.lang.get())
                .to_owned();

            match s.get::<HyUuid>("id") {
                Ok(x) => {
                    let mut perm = match &x {
                        Some(x) => {
                            let mut perm = if x.is_nil() {
                                // root
                                HashMap::from([(
                                    ROOT_ID,
                                    PermissionItem {
                                        name: "root".to_owned(),
                                        pid: ROOT_ID,
                                        perm: PERM_ALL,
                                        ..Default::default()
                                    },
                                )])
                            } else {
                                let tx = db.begin().await.unwrap();
                                let perm = skynet
                                    .get_user_perm(&tx, x)
                                    .await
                                    .unwrap()
                                    .into_iter()
                                    .map(|x| (x.pid, x))
                                    .collect();
                                tx.commit().await.unwrap();
                                perm
                            };
                            perm.insert(
                                USER_ID,
                                PermissionItem {
                                    name: "user".to_owned(),
                                    pid: USER_ID,
                                    perm: PERM_ALL,
                                    ..Default::default()
                                },
                            );
                            perm
                        }
                        None => HashMap::new(),
                    };
                    perm.insert(
                        GUEST_ID,
                        PermissionItem {
                            name: "guest".to_owned(),
                            pid: GUEST_ID,
                            perm: PERM_ALL,
                            ..Default::default()
                        },
                    );
                    let mut ext = req.extensions_mut();
                    let ext = ext.get_mut::<RequestExtension>().unwrap();
                    ext.lang = lang.clone();
                    Ok(Self {
                        uid: x,
                        username: s.get::<String>("name")?,
                        ip: get_real_ip(&req, skynet),
                        start_time: ext.start_time,
                        lang: ext.lang.clone(),
                        request_id: ext.request_id,
                        perm,
                    })
                }
                Err(e) => Err(e.into()),
            }
        })
    }
}

pub trait ResponseCodeTrait {
    fn code(&self) -> u32;
    fn translate(&self, skynet: &Skynet, locale: &str) -> String;
}

#[derive(Debug, Clone, Copy)]
#[repr(u32)]
pub enum ResponseCode {
    CodeOK,
    CodeNotReady,
    CodeRecaptchaInvalid,
    CodeUserInvalid,
    CodeUserNotExist,
    CodeUserExist,
    CodeUserRoot,
    CodeUserInvalidAvatar,
    CodeGroupNotExist,
    CodeGroupExist,
    CodePermissionNotExist,
    CodePluginLoaded,
    CodePluginExist,
    CodePluginInvalid,
    CodePluginInvalidHash,
}

impl ResponseCodeTrait for ResponseCode {
    fn translate(&self, skynet: &Skynet, locale: &str) -> String {
        t!(
            skynet,
            match self {
                Self::CodeOK => "response.success",
                Self::CodeNotReady => "response.not_ready",
                Self::CodeRecaptchaInvalid => "response.recaptcha.invalid",
                Self::CodeUserInvalid => "response.user.invalid",
                Self::CodeUserNotExist => "response.user.notexist",
                Self::CodeUserExist => "response.user.exist",
                Self::CodeUserRoot => "response.user.root",
                Self::CodeUserInvalidAvatar => "response.user.invalid_avatar",
                Self::CodeGroupNotExist => "response.group.notexist",
                Self::CodeGroupExist => "response.group.exist",
                Self::CodePermissionNotExist => "response.permission.notexist",
                Self::CodePluginLoaded => "response.plugin.loaded",
                Self::CodePluginExist => "response.plugin.exist",
                Self::CodePluginInvalid => "response.plugin.invalid",
                Self::CodePluginInvalidHash => "response.plugin.invalid_hash",
            },
            locale
        )
    }

    fn code(&self) -> u32 {
        *self as u32
    }
}

#[derive(Debug, Derivative)]
#[derivative(Default)]
pub struct ResponseCookie {
    pub name: String,
    pub value: String,
    #[derivative(Default(value = "\"/\".to_owned()"))]
    pub path: String,
    pub max_age: Duration,
    #[derivative(Default(value = "true"))]
    pub http_only: bool,
    #[derivative(Default(value = "SameSite::Strict"))]
    pub same_site: SameSite,
    #[derivative(Default(value = "true"))]
    pub secure: bool,
}

#[must_use]
#[derive(Serialize, Derivative)]
#[derivative(Default)]
pub struct Response<'a> {
    #[serde(skip)]
    #[derivative(Default(value = "Some(Box::new(ResponseCode::CodeOK))"))]
    inner: Option<Box<dyn ResponseCodeTrait>>,

    #[serde(skip)]
    #[derivative(Default(value = "200"))]
    pub http_code: u16,
    pub code: u32,
    pub msg: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<serde_json::Value>,

    #[serde(skip)]
    pub add_cookies: Vec<Cookie<'a>>,
}

impl<'a> Response<'a> {
    pub fn ok() -> Self {
        Self::default()
    }

    pub fn new<C>(code: C) -> Self
    where
        C: ResponseCodeTrait + 'static,
    {
        Self {
            inner: Some(Box::new(code)),
            ..Default::default()
        }
    }

    pub fn bad_request<S: AsRef<str>>(s: S) -> Self {
        Self {
            http_code: 400,
            inner: None,
            msg: s.as_ref().to_owned(),
            ..Default::default()
        }
    }

    pub fn not_found() -> Self {
        Self {
            http_code: 404,
            ..Default::default()
        }
    }

    pub fn add_cookie(mut self, c: ResponseCookie) -> Self {
        self.add_cookies.push(
            Cookie::build(c.name, c.value)
                .path(c.path)
                .max_age(c.max_age)
                .http_only(c.http_only)
                .same_site(c.same_site)
                .secure(c.secure)
                .finish(),
        );
        self
    }

    pub fn data<T: Serialize>(data: T) -> Self {
        Self {
            data: Some(json!(data)),
            ..Default::default()
        }
    }
}

impl<'a> Responder for Response<'a> {
    type Body = EitherBody<String>;

    fn respond_to(mut self, req: &HttpRequest) -> HttpResponse<Self::Body> {
        let skynet = req.app_data::<Data<Skynet>>().unwrap();
        if let Some(x) = &self.inner {
            self.msg = x.translate(
                skynet,
                &req.extensions().get::<RequestExtension>().unwrap().lang,
            );
            self.code = x.code();
        }
        let mut rsp = if self.http_code == 200 {
            let body = serde_json::to_string(&self).unwrap();
            HttpResponse::build(StatusCode::from_u16(self.http_code).unwrap())
                .content_type(ContentType::json())
                .message_body(body)
                .unwrap()
                .map_into_left_body()
        } else {
            HttpResponse::build(StatusCode::from_u16(self.http_code).unwrap())
                .message_body(self.msg)
                .unwrap()
                .map_into_left_body()
        };
        for i in self.add_cookies {
            rsp.add_cookie(&i).unwrap();
        }
        rsp
    }
}

#[derive(Debug)]
pub struct RspError(anyhow::Error);

impl Display for RspError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0.to_string())
    }
}

impl ResponseError for RspError {
    fn status_code(&self) -> StatusCode {
        StatusCode::INTERNAL_SERVER_ERROR
    }

    fn error_response(&self) -> HttpResponse {
        HttpResponse::build(self.status_code()).finish()
    }
}

impl<T> From<T> for RspError
where
    T: Into<anyhow::Error>,
{
    fn from(t: T) -> Self {
        Self(t.into())
    }
}

pub type RspResult<T> = Result<T, RspError>;
