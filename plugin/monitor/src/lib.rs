use agent_ws::WSAddr;
use futures::executor::block_on;
use migration::migrator::Migrator;
use monitor_service::{PluginSrv, ID};
use parking_lot::RwLock;
use skynet::{
    actix_web::web::{delete, get, post, put},
    anyhow, async_trait, create_plugin,
    log::{info, warn},
    permission::{IDTypes::PermManagePluginID, PermEntry, PERM_READ, PERM_WRITE},
    plugin::{self, Plugin},
    request::{APIRoute, PermType},
    sea_orm::{DatabaseConnection, TransactionTrait},
    utils,
    uuid::uuid,
    HyUuid, MenuItem, Result, Skynet,
};
use skynet_i18n::i18n;
use std::{
    collections::HashMap,
    path::PathBuf,
    sync::{Arc, OnceLock},
};

mod agent_session;
mod agent_ws;
mod api;
mod migration;
mod request;
mod web_ws;

static SERVICE: OnceLock<Arc<PluginSrv>> = OnceLock::new();
static DB: OnceLock<DatabaseConnection> = OnceLock::new();
static ADDRESS: OnceLock<RwLock<HashMap<HyUuid, WSAddr>>> = OnceLock::new();

#[derive(Debug, Default)]
struct Monitor;

#[async_trait]
impl Plugin for Monitor {
    fn on_load(&self, _: PathBuf, mut skynet: Skynet) -> (Skynet, Result<()>) {
        if !skynet.shared_api.contains_key(&agent_service::ID) {
            warn!("Agent plugin not enabled, auto update disabled");
        }
        let db = match plugin::init_db(&skynet, Migrator {}) {
            Ok(db) => db,
            Err(e) => return (skynet, Err(e)),
        };
        let _ = DB.set(db);

        let mut srv = PluginSrv {
            manage_id: skynet.default_id[PermManagePluginID],
            ..Default::default()
        };
        if let Err(e) = block_on(async {
            let tx = DB.get().unwrap().begin().await?;
            if PluginSrv::get_token(&skynet).is_none() {
                info!("Token not found, generating new one");
                PluginSrv::set_token(&tx, &skynet, &utils::rand_string(32)).await?;
            }
            srv.view_id = skynet
                .perm
                .find_or_init(&tx, &format!("view.plugin.{ID}"), "plugin monitor viewer")
                .await?
                .id;
            srv.init(&tx).await?;
            tx.commit().await.map_err(anyhow::Error::from)
        }) {
            return (skynet, Err(e));
        }

        // DB committed. Cannot return err below.
        skynet.insert_menu(
            MenuItem {
                id: HyUuid(uuid!("f47a0d3a-f09e-4e5d-b62c-0012225e5155")),
                name: format!("{ID}.menu.monitor"),
                path: format!("/plugin/{ID}/config"),
                perm: Some(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                }),
                ..Default::default()
            },
            1,
            Some(HyUuid(uuid!("cca5b3b0-40a3-465c-8b08-91f3e8d3b14d"))),
        );
        skynet.insert_menu(
            MenuItem {
                id: HyUuid(uuid!("d2231000-53be-46ac-87ae-73fb3f76f18f")),
                name: format!("{ID}.menu.monitor"),
                path: format!("/plugin/{ID}/view"),
                perm: Some(PermEntry {
                    pid: srv.view_id,
                    perm: PERM_READ,
                }),
                ..Default::default()
            },
            0,
            Some(HyUuid(uuid!("d00d36d0-6068-4447-ab04-f82ce893c04e"))),
        );
        skynet.add_locale(i18n!("locales"));
        let _ = SERVICE.set(Arc::new(srv));
        let _ = ADDRESS.set(RwLock::new(HashMap::new()));
        skynet
            .shared_api
            .insert(ID, Box::new(SERVICE.get().unwrap().to_owned()));
        (skynet, Ok(()))
    }

    fn on_route(&self, skynet: &Skynet, mut r: Vec<APIRoute>) -> Vec<APIRoute> {
        r.extend(vec![
            APIRoute {
                path: format!("/plugins/{ID}/ws"),
                route: get().to(web_ws::service),
                permission: PermType::Entry(PermEntry {
                    pid: SERVICE.get().unwrap().view_id,
                    perm: PERM_WRITE,
                }),
                ws_csrf: true,
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents/ws"),
                route: get().to(agent_ws::service),
                permission: PermType::Entry(PermEntry::new_guest()),
                ..Default::default()
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents"),
                route: get().to(api::get_agents),
                permission: PermType::Custom(Box::new(|x| {
                    PermEntry {
                        pid: SERVICE.get().unwrap().view_id,
                        perm: PERM_READ,
                    }
                    .check(x)
                        || PermEntry {
                            pid: SERVICE.get().unwrap().manage_id,
                            perm: PERM_READ,
                        }
                        .check(x)
                })),
                ..Default::default()
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents"),
                route: delete().to(api::delete_agents),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                }),
                ..Default::default()
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents/{{aid}}"),
                route: put().to(api::put_agent),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                }),
                ..Default::default()
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents/{{aid}}"),
                route: delete().to(api::delete_agent),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                }),
                ..Default::default()
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents/{{aid}}/reconnect"),
                route: post().to(api::reconnect_agent),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                }),
                ..Default::default()
            },
            APIRoute {
                path: format!("/plugins/{ID}/settings"),
                route: get().to(api::get_settings),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                }),
                ..Default::default()
            },
            APIRoute {
                path: format!("/plugins/{ID}/settings"),
                route: put().to(api::put_settings),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                }),
                ..Default::default()
            },
        ]);
        r
    }
}

create_plugin!(Monitor, Monitor::default);
