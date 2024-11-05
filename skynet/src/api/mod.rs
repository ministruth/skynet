use actix_cloud::{actix_web::http::Method, router::CSRFType};
use enum_map::EnumMap;
use skynet_api::{
    permission::{
        IDTypes::{self, *},
        PermEntry, PermType, PERM_READ, PERM_WRITE,
    },
    request::{box_json_router, Router},
    uuid, HyUuid, MenuItem, Skynet,
};

mod auth;
mod group;
mod misc;
mod notifications;
mod permission;
mod plugin;
mod setting;
mod user;

pub fn new_menu(id: &EnumMap<IDTypes, HyUuid>) -> Vec<MenuItem> {
    vec![
        MenuItem {
            id: HyUuid(uuid!("7bd9a6d3-db3d-4954-89ca-d5b1f3d9974f")),
            name: String::from("menu.dashboard"),
            path: String::from("/dashboard"),
            icon: String::from("DashboardOutlined"),
            perm: Some(PermEntry::new_user()),
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("d00d36d0-6068-4447-ab04-f82ce893c04e")),
            name: String::from("menu.service"),
            icon: String::from("FunctionOutlined"),
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("cca5b3b0-40a3-465c-8b08-91f3e8d3b14d")),
            name: String::from("menu.plugin"),
            icon: String::from("ApiOutlined"),
            children: vec![MenuItem {
                id: HyUuid(uuid!("251a16e1-655b-4716-8766-cd2bc66d6309")),
                name: String::from("menu.plugin.manage"),
                path: String::from("/plugin"),
                perm: Some(PermEntry {
                    pid: id[PermManagePluginID],
                    perm: PERM_READ,
                }),
                ..Default::default()
            }],
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("4d6c60d7-9c2a-44f0-b85a-346425df792f")),
            name: String::from("menu.user"),
            omit_empty: true,
            icon: String::from("UserOutlined"),
            children: vec![
                MenuItem {
                    id: HyUuid(uuid!("0d2165b9-e08b-429f-ad4e-420472083e0f")),
                    name: String::from("menu.user.user"),
                    path: String::from("/user"),
                    perm: Some(PermEntry {
                        pid: id[PermManageUserID],
                        perm: PERM_READ,
                    }),
                    ..Default::default()
                },
                MenuItem {
                    id: HyUuid(uuid!("03e3caeb-9008-4e5c-9e19-c11d6b567aa7")),
                    name: String::from("menu.user.group"),
                    path: String::from("/group"),
                    perm: Some(PermEntry {
                        pid: id[PermManageUserID],
                        perm: PERM_READ,
                    }),
                    ..Default::default()
                },
            ],
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("06c21cbc-b43f-4b43-a633-8baf2221493f")),
            name: String::from("menu.notification"),
            path: String::from("/notification"),
            icon: String::from("NotificationOutlined"),
            perm: Some(PermEntry {
                pid: id[PermManageNotificationID],
                perm: PERM_READ,
            }),
            badge_func: Some(Box::new(|s: &Skynet| -> i64 {
                s.logger.get_unread().try_into().unwrap()
            })),
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("4b9df963-c540-48f4-9bfb-500f06ecfef0")),
            name: String::from("menu.system"),
            path: String::from("/system"),
            icon: String::from("SettingOutlined"),
            perm: Some(PermEntry {
                pid: id[PermManageSystemID],
                perm: PERM_READ,
            }),
            ..Default::default()
        },
    ]
}

pub fn new_api(id: &EnumMap<IDTypes, HyUuid>, disable_csrf: bool) -> Vec<Router> {
    let csrf = if disable_csrf {
        CSRFType::Disabled
    } else {
        CSRFType::Header
    };
    vec![
        Router {
            path: String::from("/settings/public"),
            method: Method::GET,
            route: box_json_router(setting::get_public),
            checker: PermType::Entry(PermEntry::new_guest()),
            csrf,
        },
        Router {
            path: String::from("/signin"),
            method: Method::POST,
            route: box_json_router(auth::signin),
            checker: PermType::Entry(PermEntry::new_guest()),
            csrf,
        },
        Router {
            path: String::from("/signout"),
            method: Method::POST,
            route: box_json_router(auth::signout),
            checker: PermType::Entry(PermEntry::new_user()),
            csrf,
        },
        Router {
            path: String::from("/health"),
            method: Method::GET,
            route: box_json_router(misc::health),
            checker: PermType::Entry(PermEntry::new_guest()),
            csrf,
        },
        Router {
            path: String::from("/access"),
            method: Method::GET,
            route: box_json_router(auth::get_access),
            checker: PermType::Entry(PermEntry::new_guest()),
            csrf,
        },
        Router {
            path: String::from("/token"),
            method: Method::GET,
            route: box_json_router(auth::get_token),
            checker: PermType::Entry(PermEntry::new_guest()),
            csrf,
        },
        Router {
            path: String::from("/shutdown"),
            method: Method::POST,
            route: box_json_router(misc::shutdown),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageSystemID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/menus"),
            method: Method::GET,
            route: box_json_router(misc::get_menus),
            checker: PermType::Entry(PermEntry::new_user()),
            csrf,
        },
        Router {
            path: String::from("/dashboard"),
            method: Method::GET,
            route: box_json_router(misc::get_menus),
            checker: PermType::Entry(PermEntry::new_user()),
            csrf,
        },
        Router {
            path: String::from("/users"),
            method: Method::GET,
            route: box_json_router(user::get_all),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/users"),
            method: Method::POST,
            route: box_json_router(user::add),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/users"),
            method: Method::DELETE,
            route: box_json_router(user::delete_batch),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/users/{uid}"),
            method: Method::GET,
            route: box_json_router(user::get),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/users/{uid}"),
            method: Method::PUT,
            route: box_json_router(user::put),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/users/{uid}"),
            method: Method::DELETE,
            route: box_json_router(user::delete),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/users/{uid}/kick"),
            method: Method::POST,
            route: box_json_router(user::kick),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/users/{uid}/groups"),
            method: Method::GET,
            route: box_json_router(user::get_group),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/users/{uid}/permissions"),
            method: Method::GET,
            route: box_json_router(permission::get_user),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/users/{uid}/permissions"),
            method: Method::PUT,
            route: box_json_router(permission::put_user),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups"),
            method: Method::GET,
            route: box_json_router(group::get_all),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups"),
            method: Method::POST,
            route: box_json_router(group::add),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups"),
            method: Method::DELETE,
            route: box_json_router(group::delete_batch),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups/{gid}"),
            method: Method::GET,
            route: box_json_router(group::get),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups/{gid}"),
            method: Method::PUT,
            route: box_json_router(group::put),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups/{gid}"),
            method: Method::DELETE,
            route: box_json_router(group::delete),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups/{gid}/users"),
            method: Method::GET,
            route: box_json_router(group::get_user),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups/{gid}/users"),
            method: Method::POST,
            route: box_json_router(group::add_user),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups/{gid}/users"),
            method: Method::DELETE,
            route: box_json_router(group::delete_user_batch),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups/{gid}/users/{uid}"),
            method: Method::DELETE,
            route: box_json_router(group::delete_user),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups/{gid}/permissions"),
            method: Method::GET,
            route: box_json_router(permission::get_group),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/groups/{gid}/permissions"),
            method: Method::PUT,
            route: box_json_router(permission::put_group),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/permissions"),
            method: Method::GET,
            route: box_json_router(permission::get),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageUserID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/notifications"),
            method: Method::GET,
            route: box_json_router(notifications::get_all),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageNotificationID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/notifications"),
            method: Method::DELETE,
            route: box_json_router(notifications::delete_all),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageNotificationID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/notifications/unread"),
            method: Method::GET,
            route: box_json_router(notifications::get_unread),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManageNotificationID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/plugins"),
            method: Method::GET,
            route: box_json_router(plugin::get),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManagePluginID],
                perm: PERM_READ,
            }),
            csrf,
        },
        Router {
            path: String::from("/plugins/{id}"),
            method: Method::PUT,
            route: box_json_router(plugin::put),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManagePluginID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/plugins/{id}"),
            method: Method::DELETE,
            route: box_json_router(plugin::delete),
            checker: PermType::Entry(PermEntry {
                pid: id[PermManagePluginID],
                perm: PERM_WRITE,
            }),
            csrf,
        },
        Router {
            path: String::from("/plugins/entries"),
            method: Method::GET,
            route: box_json_router(plugin::get_entries),
            checker: PermType::Entry(PermEntry::new_guest()),
            csrf,
        },
    ]
}
