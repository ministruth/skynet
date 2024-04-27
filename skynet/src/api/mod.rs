#![allow(clippy::enum_glob_use)]
use actix_http::{body::EitherBody, HttpMessage, StatusCode};
use actix_web::{
    dev::{forward_ready, Service, ServiceRequest, ServiceResponse, Transform},
    web::{self, delete, get, post, put},
    Error, HttpResponse,
};
use anyhow::Result;
use derivative::Derivative;
use futures::future::LocalBoxFuture;
use log::{debug, error};
use qstring::QString;
use redis::{aio::ConnectionManager, AsyncCommands};
use serde_json::json;
use skynet::{
    permission::{IDTypes::*, PermEntry, PERM_READ, PERM_WRITE},
    request::{APIRoute, PermType, Request, RspError},
    utils, HyUuid, MenuItem, Skynet,
};
use std::rc::Rc;
use std::{
    fmt::Debug,
    future::{ready, Ready},
};
use thiserror::Error;
use uuid::uuid;

pub mod auth;
pub mod group;
pub mod misc;
pub mod notifications;
pub mod permission;
pub mod plugin;
pub mod setting;
pub mod user;

pub const CSRF_COOKIE: &str = "CSRF_TOKEN";
pub const CSRF_HEADER: &str = "X-CSRF-Token";

pub fn new_menu(skynet: &Skynet) -> Vec<MenuItem> {
    vec![
        MenuItem {
            id: HyUuid(uuid!("7bd9a6d3-db3d-4954-89ca-d5b1f3d9974f")),
            name: "menu.dashboard".to_owned(),
            path: "/dashboard".to_owned(),
            icon: "DashboardOutlined".to_owned(),
            perm: Some(PermEntry::new_user()),
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("d00d36d0-6068-4447-ab04-f82ce893c04e")),
            name: "menu.service".to_owned(),
            icon: "FunctionOutlined".to_owned(),
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("cca5b3b0-40a3-465c-8b08-91f3e8d3b14d")),
            name: "menu.plugin".to_owned(),
            icon: "ApiOutlined".to_owned(),
            children: vec![MenuItem {
                id: HyUuid(uuid!("251a16e1-655b-4716-8766-cd2bc66d6309")),
                name: "menu.plugin.manage".to_owned(),
                path: "/plugin".to_owned(),
                perm: Some(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                }),
                ..Default::default()
            }],
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("4d6c60d7-9c2a-44f0-b85a-346425df792f")),
            name: "menu.user".to_owned(),
            omit_empty: true,
            icon: "UserOutlined".to_owned(),
            children: vec![
                MenuItem {
                    id: HyUuid(uuid!("0d2165b9-e08b-429f-ad4e-420472083e0f")),
                    name: "menu.user.user".to_owned(),
                    path: "/user".to_owned(),
                    perm: Some(PermEntry {
                        pid: skynet.default_id[PermManageUserID],
                        perm: PERM_READ,
                    }),
                    ..Default::default()
                },
                MenuItem {
                    id: HyUuid(uuid!("03e3caeb-9008-4e5c-9e19-c11d6b567aa7")),
                    name: "menu.user.group".to_owned(),
                    path: "/group".to_owned(),
                    perm: Some(PermEntry {
                        pid: skynet.default_id[PermManageUserID],
                        perm: PERM_READ,
                    }),
                    ..Default::default()
                },
            ],
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("06c21cbc-b43f-4b43-a633-8baf2221493f")),
            name: "menu.notification".to_owned(),
            path: "/notification".to_owned(),
            icon: "NotificationOutlined".to_owned(),
            perm: Some(PermEntry {
                pid: skynet.default_id[PermManageNotificationID],
                perm: PERM_READ,
            }),
            badge_func: Some(Box::new(|s: &Skynet| -> i64 {
                s.get_unread().try_into().unwrap()
            })),
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("4b9df963-c540-48f4-9bfb-500f06ecfef0")),
            name: "menu.system".to_owned(),
            path: "/system".to_owned(),
            icon: "SettingOutlined".to_owned(),
            perm: Some(PermEntry {
                pid: skynet.default_id[PermManageSystemID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
    ]
}

#[allow(clippy::module_name_repetitions, clippy::too_many_lines)]
pub fn new_api(skynet: &Skynet) -> Vec<APIRoute> {
    vec![
        APIRoute {
            path: "/settings/public".to_owned(),
            route: get().to(setting::get_public),
            permission: PermType::Entry(PermEntry::new_guest()),
            ..Default::default()
        },
        APIRoute {
            path: "/signin".to_owned(),
            route: post().to(auth::signin),
            permission: PermType::Entry(PermEntry::new_guest()),
            ..Default::default()
        },
        APIRoute {
            path: "/signout".to_owned(),
            route: post().to(auth::signout),
            permission: PermType::Entry(PermEntry::new_user()),
            ..Default::default()
        },
        APIRoute {
            path: "/ping".to_owned(),
            route: get().to(misc::ping),
            permission: PermType::Entry(PermEntry::new_guest()),
            ..Default::default()
        },
        APIRoute {
            path: "/access".to_owned(),
            route: get().to(auth::get_access),
            permission: PermType::Entry(PermEntry::new_guest()),
            ..Default::default()
        },
        APIRoute {
            path: "/token".to_owned(),
            route: get().to(auth::get_token),
            permission: PermType::Entry(PermEntry::new_guest()),
            ..Default::default()
        },
        APIRoute {
            path: "/shutdown".to_owned(),
            route: post().to(misc::shutdown),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageSystemID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/menus".to_owned(),
            route: get().to(misc::get_menus),
            permission: PermType::Entry(PermEntry::new_user()),
            ..Default::default()
        },
        APIRoute {
            path: "/users".to_owned(),
            route: get().to(user::get_all),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/users".to_owned(),
            route: post().to(user::add),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/users".to_owned(),
            route: delete().to(user::delete_batch),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/users/{uid}".to_owned(),
            route: get().to(user::get),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/users/{uid}".to_owned(),
            route: put().to(user::put),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/users/{uid}".to_owned(),
            route: delete().to(user::delete),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/users/{uid}/kick".to_owned(),
            route: post().to(user::kick),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/users/{uid}/groups".to_owned(),
            route: get().to(user::get_group),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/users/{uid}/permissions".to_owned(),
            route: get().to(permission::get_user),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/users/{uid}/permissions".to_owned(),
            route: put().to(permission::put_user),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups".to_owned(),
            route: get().to(group::get_all),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups".to_owned(),
            route: post().to(group::add),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups".to_owned(),
            route: delete().to(group::delete_batch),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups/{gid}".to_owned(),
            route: get().to(group::get),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups/{gid}".to_owned(),
            route: put().to(group::put),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups/{gid}".to_owned(),
            route: delete().to(group::delete),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups/{gid}/users".to_owned(),
            route: get().to(group::get_user),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups/{gid}/users".to_owned(),
            route: post().to(group::add_user),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups/{gid}/users".to_owned(),
            route: delete().to(group::delete_user_batch),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups/{gid}/users/{uid}".to_owned(),
            route: delete().to(group::delete_user),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups/{gid}/permissions".to_owned(),
            route: get().to(permission::get_group),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/groups/{gid}/permissions".to_owned(),
            route: put().to(permission::put_group),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/permissions".to_owned(),
            route: get().to(permission::get),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageUserID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/notifications".to_owned(),
            route: get().to(notifications::get_all),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageNotificationID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/notifications".to_owned(),
            route: delete().to(notifications::delete_all),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageNotificationID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/notifications/unread".to_owned(),
            route: get().to(notifications::get_unread),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManageNotificationID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/plugins".to_owned(),
            route: get().to(plugin::get),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManagePluginID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/plugins/{id}".to_owned(),
            route: put().to(plugin::put),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManagePluginID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/plugins/{id}".to_owned(),
            route: delete().to(plugin::delete),
            permission: PermType::Entry(PermEntry {
                pid: skynet.default_id[PermManagePluginID],
                perm: PERM_WRITE,
            }),
            ..Default::default()
        },
        APIRoute {
            path: "/plugins/entries".to_owned(),
            route: get().to(plugin::get_entries),
            permission: PermType::Entry(PermEntry::new_guest()),
            ..Default::default()
        },
    ]
}

pub fn router(api: Vec<APIRoute>, disable_csrf: bool) -> Box<dyn FnOnce(&mut web::ServiceConfig)> {
    Box::new(move |cfg| {
        for i in api {
            if !i.path.is_empty() {
                cfg.route(
                    &i.path,
                    i.route.wrap(SkynetGuard {
                        permission: Rc::new(i.permission),
                        ws_csrf: i.ws_csrf,
                        disable_csrf,
                    }),
                );
            }
        }
    })
}

#[derive(Error, Derivative)]
#[derivative(Debug)]
pub enum APIError {
    #[error("Validation error: missing field `{0}`")]
    MissingField(String),
}

pub struct SkynetGuard {
    permission: Rc<PermType>,
    disable_csrf: bool,
    ws_csrf: bool,
}

impl<S: 'static, B> Transform<S, ServiceRequest> for SkynetGuard
where
    S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = Error>,
    S::Future: 'static,
    B: 'static + Debug,
{
    type Response = ServiceResponse<EitherBody<B>>;
    type Error = Error;
    type InitError = ();
    type Transform = SkynetGuardMiddleware<S>;
    type Future = Ready<Result<Self::Transform, Self::InitError>>;

    fn new_transform(&self, service: S) -> Self::Future {
        ready(Ok(SkynetGuardMiddleware {
            service: Rc::new(service),
            permission: self.permission.clone(),
            disable_csrf: self.disable_csrf,
            ws_csrf: self.ws_csrf,
        }))
    }
}

pub struct SkynetGuardMiddleware<S> {
    service: Rc<S>,
    permission: Rc<PermType>,
    disable_csrf: bool,
    ws_csrf: bool,
}

impl<S, B> SkynetGuardMiddleware<S>
where
    S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = Error> + 'static,
    S::Future: 'static,
    B: 'static,
{
    fn get_safe_header(req: &ServiceRequest, name: &str) -> Option<String> {
        let mut ret: Vec<&str> = req
            .headers()
            .get_all(name)
            .map(|x| x.to_str().unwrap())
            .collect();
        if ret.len() != 1 {
            return None;
        }
        ret.pop().map(ToOwned::to_owned)
    }

    async fn check_csrf(req: &ServiceRequest, allow_param: bool) -> Result<bool, actix_web::Error> {
        let skynet = req.app_data::<web::Data<Skynet>>().unwrap();
        let redis = req.app_data::<web::Data<ConnectionManager>>().unwrap();
        let cookie = req.cookie(CSRF_COOKIE);
        if cookie.is_none() {
            debug!("Missing CSRF cookie");
            return Ok(false);
        }
        let mut csrf = Self::get_safe_header(req, CSRF_HEADER);
        if csrf.is_none() && allow_param {
            let qs = QString::from(req.query_string());
            csrf = qs.get(CSRF_HEADER).map(ToOwned::to_owned);
        }
        if csrf.is_none() {
            debug!("Incorrect CSRF header");
            return Ok(false);
        }
        let csrf = csrf.unwrap();
        if csrf != cookie.unwrap().value() {
            debug!("Mismatch CSRF cookie and header");
            return Ok(false);
        }
        let result = check_csrf_token(skynet, redis, &csrf)
            .await
            .map_err(RspError::from)?;
        if !result {
            debug!("Broken CSRF header");
        }
        Ok(result)
    }
}

impl<S, B> Service<ServiceRequest> for SkynetGuardMiddleware<S>
where
    S: Service<ServiceRequest, Response = ServiceResponse<B>, Error = Error> + 'static,
    S::Future: 'static,
    B: 'static + Debug,
{
    type Response = ServiceResponse<EitherBody<B>>;
    type Error = Error;
    type Future = LocalBoxFuture<'static, Result<Self::Response, Self::Error>>;

    forward_ready!(service);

    fn call(&self, mut req: ServiceRequest) -> Self::Future {
        let srv = self.service.clone();
        let permission = self.permission.clone();
        let disable_csrf = self.disable_csrf;
        let ws_csrf = self.ws_csrf;

        Box::pin(async move {
            if !disable_csrf
                && (!req.method().is_safe() || ws_csrf)
                && !Self::check_csrf(&req, ws_csrf).await?
            {
                return Ok(
                    req.into_response(HttpResponse::BadRequest().finish().map_into_right_body())
                );
            }
            let r = req.extract::<Request>().await?;
            let result = match &*permission {
                PermType::Entry(x) => x.check(&r.perm),
                PermType::Custom(x) => x(&r.perm),
            };
            req.extensions_mut().insert(r); // cache existing value
            if result {
                let request = req.request().clone();
                let rsp = srv.call(req).await?;
                if rsp.status() == StatusCode::BAD_REQUEST {
                    debug!(target: "skynet-web","{:?}",rsp.into_body());
                    Ok(ServiceResponse::new(
                        request,
                        HttpResponse::BadRequest().finish().map_into_right_body(),
                    ))
                } else if rsp.status() == StatusCode::INTERNAL_SERVER_ERROR {
                    if let Some(e) = rsp.response().error() {
                        error!(
                            target: "skynet-web",
                            "Error handle request\n{}",
                            json!({
                                "method": rsp.request().method().as_str(),
                                "path": rsp.request().path(),
                                "error": e.to_string(),
                            })
                        );
                    }
                    Ok(rsp.map_into_left_body())
                } else if rsp.status() == StatusCode::NOT_FOUND {
                    Ok(ServiceResponse::new(
                        request,
                        HttpResponse::NotFound().finish().map_into_right_body(),
                    ))
                } else {
                    Ok(rsp.map_into_left_body())
                }
            } else {
                Ok(req.into_response(HttpResponse::Forbidden().finish().map_into_right_body()))
            }
        })
    }
}

/// Generate new csrf token.
/// 32 length, a-zA-Z0-9.
pub async fn new_csrf_token(s: &Skynet, redis: &ConnectionManager) -> Result<String> {
    let token = utils::rand_string(32);
    redis
        .clone()
        .set_ex(
            format!("{}{}", s.config.csrf_prefix.get(), token),
            "1",
            s.config.csrf_timeout.get().try_into().unwrap(),
        )
        .await?;
    Ok(token)
}

/// Check csrf token.
pub async fn check_csrf_token(s: &Skynet, redis: &ConnectionManager, token: &str) -> Result<bool> {
    let res: Option<String> = redis
        .clone()
        .get_del(format!("{}{}", s.config.csrf_prefix.get(), token))
        .await?;
    res.map_or_else(|| Ok(false), |x| Ok(x == "1"))
}
