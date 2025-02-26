use actix_cloud::{actix_web::Route, router::CSRFType};
use enum_map::EnumMap;
use skynet_api::{
    HyUuid, MenuItem, Skynet,
    permission::{
        IDTypes::{self, *},
        PERM_READ, PERM_WRITE, PermChecker, PermEntry,
    },
    request::{Method, Router, RouterType::Inner},
    uuid,
};

use crate::logger;

mod auth;
mod client;
mod dashboard;
mod group;
mod misc;
mod notifications;
mod permission;
mod plugin;
mod setting;
mod user;

pub fn set_menu_badge(skynet: &mut Skynet) {
    if !skynet.reset_menu_badge(
        HyUuid(uuid!("06c21cbc-b43f-4b43-a633-8baf2221493f")),
        logger::UNREAD.clone(),
    ) {
        panic!("Set menu badge fail");
    }
}

pub fn new_menu(id: &EnumMap<IDTypes, HyUuid>) -> Vec<MenuItem> {
    vec![
        MenuItem {
            id: HyUuid(uuid!("7bd9a6d3-db3d-4954-89ca-d5b1f3d9974f")),
            name: String::from("menu.dashboard"),
            path: String::from("/dashboard"),
            icon: String::from("DashboardOutlined"),
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("6b2cfbf9-f598-4bc2-a940-42ec673df5d1")),
            name: String::from("menu.account"),
            path: String::from("/account"),
            icon: String::from("UserOutlined"),
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
                checker: PermChecker::new_entry(id[PermManagePluginID], PERM_READ),
                ..Default::default()
            }],
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("4d6c60d7-9c2a-44f0-b85a-346425df792f")),
            name: String::from("menu.user"),
            omit_empty: true,
            icon: String::from("TeamOutlined"),
            children: vec![
                MenuItem {
                    id: HyUuid(uuid!("0d2165b9-e08b-429f-ad4e-420472083e0f")),
                    name: String::from("menu.user.user"),
                    path: String::from("/user"),
                    checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
                    ..Default::default()
                },
                MenuItem {
                    id: HyUuid(uuid!("03e3caeb-9008-4e5c-9e19-c11d6b567aa7")),
                    name: String::from("menu.user.group"),
                    path: String::from("/group"),
                    checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
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
            checker: PermChecker::new_entry(id[PermManageNotificationID], PERM_READ),
            ..Default::default()
        },
        MenuItem {
            id: HyUuid(uuid!("4b9df963-c540-48f4-9bfb-500f06ecfef0")),
            name: String::from("menu.system"),
            path: String::from("/system"),
            icon: String::from("SettingOutlined"),
            checker: PermChecker::new_entry(id[PermManageSystemID], PERM_READ),
            ..Default::default()
        },
    ]
}

pub fn api_call(name: &str, r: Route) -> Route {
    match name {
        "setting::get_public" => r.to(setting::get_public),
        "setting::get_system" => r.to(setting::get_system),
        "setting::put_system" => r.to(setting::put_system),
        "setting::reset_session_key" => r.to(setting::reset_session_key),
        "setting::reset_webpush_key" => r.to(setting::reset_webpush_key),
        "auth::signin" => r.to(auth::signin),
        "auth::signout" => r.to(auth::signout),
        "misc::health" => r.to(misc::health),
        "auth::get_access" => r.to(auth::get_access),
        "auth::get_token" => r.to(auth::get_token),
        "misc::shutdown" => r.to(misc::shutdown),
        "misc::get_menus" => r.to(misc::get_menus),
        "misc::geoip" => r.to(misc::geoip),
        "user::get_all" => r.to(user::get_all),
        "user::add" => r.to(user::add),
        "user::delete_batch" => r.to(user::delete_batch),
        "user::get_self" => r.to(user::get_self),
        "user::get" => r.to(user::get),
        "user::put_self" => r.to(user::put_self),
        "user::put" => r.to(user::put),
        "user::delete" => r.to(user::delete),
        "user::kick_self" => r.to(user::kick_self),
        "user::kick" => r.to(user::kick),
        "user::get_group" => r.to(user::get_group),
        "permission::get_user" => r.to(permission::get_user),
        "permission::put_user" => r.to(permission::put_user),
        "user::get_histories_self" => r.to(user::get_histories_self),
        "user::get_histories" => r.to(user::get_histories),
        "user::get_sessions_self" => r.to(user::get_sessions_self),
        "user::get_sessions" => r.to(user::get_sessions),
        "user::get_webpush_topics" => r.to(user::get_webpush_topics),
        "user::put_webpush_topic" => r.to(user::put_webpush_topic),
        "user::subscribe_webpush" => r.to(user::subscribe_webpush),
        "user::unsubscribe_webpush" => r.to(user::unsubscribe_webpush),
        "user::check_webpush" => r.to(user::check_webpush),
        "group::get_all" => r.to(group::get_all),
        "group::add" => r.to(group::add),
        "group::delete_batch" => r.to(group::delete_batch),
        "group::get" => r.to(group::get),
        "group::put" => r.to(group::put),
        "group::delete" => r.to(group::delete),
        "group::get_user" => r.to(group::get_user),
        "group::add_user" => r.to(group::add_user),
        "group::delete_user_batch" => r.to(group::delete_user_batch),
        "group::delete_user" => r.to(group::delete_user),
        "permission::get_group" => r.to(permission::get_group),
        "permission::put_group" => r.to(permission::put_group),
        "permission::get" => r.to(permission::get),
        "notifications::get_all" => r.to(notifications::get_all),
        "notifications::delete_all" => r.to(notifications::delete_all),
        "notifications::get_unread" => r.to(notifications::get_unread),
        "plugin::get" => r.to(plugin::get),
        "plugin::put" => r.to(plugin::put),
        "plugin::delete" => r.to(plugin::delete),
        "plugin::get_entries" => r.to(plugin::get_entries),
        "dashboard::system_info" => r.to(dashboard::system_info),
        "dashboard::runtime_info" => r.to(dashboard::runtime_info),
        _ => unreachable!(),
    }
}

pub fn new_api(id: &EnumMap<IDTypes, HyUuid>) -> Vec<Router> {
    vec![
        Router {
            path: String::from("/settings/public"),
            method: Method::Get,
            route: Inner(String::from("setting::get_public")),
            checker: PermChecker::Entry(PermEntry::new_guest()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/settings/system"),
            method: Method::Get,
            route: Inner(String::from("setting::get_system")),
            checker: PermChecker::new_entry(id[PermManageSystemID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/settings/system"),
            method: Method::Put,
            route: Inner(String::from("setting::put_system")),
            checker: PermChecker::new_entry(id[PermManageSystemID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/settings/sessionkey"),
            method: Method::Post,
            route: Inner(String::from("setting::reset_session_key")),
            checker: PermChecker::new_entry(id[PermManageSystemID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/settings/webpushkey"),
            method: Method::Post,
            route: Inner(String::from("setting::reset_webpush_key")),
            checker: PermChecker::new_entry(id[PermManageSystemID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/signin"),
            method: Method::Post,
            route: Inner(String::from("auth::signin")),
            checker: PermChecker::Entry(PermEntry::new_guest()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/signout"),
            method: Method::Post,
            route: Inner(String::from("auth::signout")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/health"),
            method: Method::Get,
            route: Inner(String::from("misc::health")),
            checker: PermChecker::Entry(PermEntry::new_guest()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/access"),
            method: Method::Get,
            route: Inner(String::from("auth::get_access")),
            checker: PermChecker::Entry(PermEntry::new_guest()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/token"),
            method: Method::Get,
            route: Inner(String::from("auth::get_token")),
            checker: PermChecker::Entry(PermEntry::new_guest()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/shutdown"),
            method: Method::Post,
            route: Inner(String::from("misc::shutdown")),
            checker: PermChecker::new_entry(id[PermManageSystemID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/menus"),
            method: Method::Get,
            route: Inner(String::from("misc::get_menus")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/geoip"),
            method: Method::Get,
            route: Inner(String::from("misc::geoip")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users"),
            method: Method::Get,
            route: Inner(String::from("user::get_all")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users"),
            method: Method::Post,
            route: Inner(String::from("user::add")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users"),
            method: Method::Delete,
            route: Inner(String::from("user::delete_batch")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self"),
            method: Method::Get,
            route: Inner(String::from("user::get_self")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/{uid}"),
            method: Method::Get,
            route: Inner(String::from("user::get")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self"),
            method: Method::Put,
            route: Inner(String::from("user::put_self")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/{uid}"),
            method: Method::Put,
            route: Inner(String::from("user::put")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/{uid}"),
            method: Method::Delete,
            route: Inner(String::from("user::delete")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self/kick"),
            method: Method::Post,
            route: Inner(String::from("user::kick_self")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/{uid}/kick"),
            method: Method::Post,
            route: Inner(String::from("user::kick")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/{uid}/groups"),
            method: Method::Get,
            route: Inner(String::from("user::get_group")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/{uid}/permissions"),
            method: Method::Get,
            route: Inner(String::from("permission::get_user")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/{uid}/permissions"),
            method: Method::Put,
            route: Inner(String::from("permission::put_user")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self/histories"),
            method: Method::Get,
            route: Inner(String::from("user::get_histories_self")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/{uid}/histories"),
            method: Method::Get,
            route: Inner(String::from("user::get_histories")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self/sessions"),
            method: Method::Get,
            route: Inner(String::from("user::get_sessions_self")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/{uid}/sessions"),
            method: Method::Get,
            route: Inner(String::from("user::get_sessions")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self/webpush"),
            method: Method::Get,
            route: Inner(String::from("user::get_webpush_topics")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self/webpush"),
            method: Method::Put,
            route: Inner(String::from("user::put_webpush_topic")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self/webpush"),
            method: Method::Post,
            route: Inner(String::from("user::subscribe_webpush")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self/webpush"),
            method: Method::Delete,
            route: Inner(String::from("user::unsubscribe_webpush")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/users/self/webpush/check"),
            method: Method::Post,
            route: Inner(String::from("user::check_webpush")),
            checker: PermChecker::Entry(PermEntry::new_user()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups"),
            method: Method::Get,
            route: Inner(String::from("group::get_all")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups"),
            method: Method::Post,
            route: Inner(String::from("group::add")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups"),
            method: Method::Delete,
            route: Inner(String::from("group::delete_batch")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups/{gid}"),
            method: Method::Get,
            route: Inner(String::from("group::get")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups/{gid}"),
            method: Method::Put,
            route: Inner(String::from("group::put")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups/{gid}"),
            method: Method::Delete,
            route: Inner(String::from("group::delete")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups/{gid}/users"),
            method: Method::Get,
            route: Inner(String::from("group::get_user")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups/{gid}/users"),
            method: Method::Post,
            route: Inner(String::from("group::add_user")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups/{gid}/users"),
            method: Method::Delete,
            route: Inner(String::from("group::delete_user_batch")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups/{gid}/users/{uid}"),
            method: Method::Delete,
            route: Inner(String::from("group::delete_user")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups/{gid}/permissions"),
            method: Method::Get,
            route: Inner(String::from("permission::get_group")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/groups/{gid}/permissions"),
            method: Method::Put,
            route: Inner(String::from("permission::put_group")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/permissions"),
            method: Method::Get,
            route: Inner(String::from("permission::get")),
            checker: PermChecker::new_entry(id[PermManageUserID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/notifications"),
            method: Method::Get,
            route: Inner(String::from("notifications::get_all")),
            checker: PermChecker::new_entry(id[PermManageNotificationID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/notifications"),
            method: Method::Delete,
            route: Inner(String::from("notifications::delete_all")),
            checker: PermChecker::new_entry(id[PermManageNotificationID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/notifications/unread"),
            method: Method::Get,
            route: Inner(String::from("notifications::get_unread")),
            checker: PermChecker::new_entry(id[PermManageNotificationID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/plugins"),
            method: Method::Get,
            route: Inner(String::from("plugin::get")),
            checker: PermChecker::new_entry(id[PermManagePluginID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/plugins/{id}"),
            method: Method::Put,
            route: Inner(String::from("plugin::put")),
            checker: PermChecker::new_entry(id[PermManagePluginID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/plugins/{id}"),
            method: Method::Delete,
            route: Inner(String::from("plugin::delete")),
            checker: PermChecker::new_entry(id[PermManagePluginID], PERM_WRITE),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/plugins/entries"),
            method: Method::Get,
            route: Inner(String::from("plugin::get_entries")),
            checker: PermChecker::Entry(PermEntry::new_guest()),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/dashboard/system_info"),
            method: Method::Get,
            route: Inner(String::from("dashboard::system_info")),
            checker: PermChecker::new_entry(id[PermManageSystemID], PERM_READ),
            csrf: CSRFType::Header,
        },
        Router {
            path: String::from("/dashboard/runtime_info"),
            method: Method::Get,
            route: Inner(String::from("dashboard::runtime_info")),
            checker: PermChecker::new_entry(id[PermManageSystemID], PERM_READ),
            csrf: CSRFType::Header,
        },
    ]
}
