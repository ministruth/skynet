use migration::migrator::Migrator;
use service::AgentSrv;
pub use service::TokenSrv;
pub mod msg;

use derivative::Derivative;
use futures::executor::block_on;
use skynet::{
    actix_web::{
        http::Method,
        web::{delete, get, put},
    },
    anyhow, async_trait, create_plugin,
    log::info,
    permission::{IDTypes::PermManagePluginID, PermEntry, PERM_READ, PERM_WRITE},
    plugin::{self, Plugin},
    request::{APIRoute, PermType},
    sea_orm::{DatabaseConnection, TransactionTrait},
    utils,
    uuid::uuid,
    HyUuid, MenuItem, Result, Skynet,
};
use skynet_i18n::i18n;
use std::sync::{Arc, OnceLock};

mod api;
mod entity;
mod migration;
mod request;
mod service;
mod ws;

static SERVICE: OnceLock<Arc<PluginSrv>> = OnceLock::new();
static ID: HyUuid = HyUuid(uuid!("2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"));

#[derive(Derivative)]
#[derivative(Default(new = "true"))]
pub struct PluginSrv {
    pub(crate) db: DatabaseConnection,
    pub agent: AgentSrv,
    pub view_id: HyUuid,
    manage_id: HyUuid,
}

#[derive(Debug, Default)]
struct Monitor;

#[async_trait]
impl Plugin for Monitor {
    fn on_load(&self, mut skynet: Skynet) -> (Skynet, Result<()>) {
        let db = match plugin::init_db(&skynet, Migrator {}) {
            Ok(db) => db,
            Err(e) => return (skynet, Err(e)),
        };

        let agent_srv = AgentSrv::new();
        let mut view_id = HyUuid::default();
        if let Err(e) = block_on(async {
            let tx = db.begin().await?;
            if TokenSrv::get(&skynet).is_none() {
                info!("Token not found, generating new one");
                TokenSrv::set(&tx, &skynet, &utils::rand_string(32)).await?;
            }
            view_id = skynet
                .perm
                .find_or_init(&tx, &format!("view.plugin.{ID}"), "plugin monitor viewer")
                .await?
                .id;
            agent_srv.init(&tx).await?;
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
        skynet.add_locale(i18n!("locales"));
        let _ = SERVICE.set(Arc::new(PluginSrv {
            agent: agent_srv,
            manage_id: skynet.default_id[PermManagePluginID],
            db,
            view_id,
        }));
        skynet
            .shared_api
            .insert(ID, Box::new(SERVICE.get().unwrap().to_owned()));
        (skynet, Ok(()))
    }

    fn on_route(&self, skynet: &Skynet, mut r: Vec<APIRoute>) -> Vec<APIRoute> {
        r.extend(vec![
            APIRoute {
                path: format!("/plugins/{ID}/ws"),
                method: Method::GET,
                route: get().to(ws::get),
                permission: PermType::Entry(PermEntry::new_guest()),
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents"),
                method: Method::GET,
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
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents"),
                method: Method::DELETE,
                route: delete().to(api::delete_agents),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                }),
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents/{{aid}}"),
                method: Method::PUT,
                route: put().to(api::put_agent),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                }),
            },
            APIRoute {
                path: format!("/plugins/{ID}/agents/{{aid}}"),
                method: Method::DELETE,
                route: delete().to(api::delete_agent),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                }),
            },
            APIRoute {
                path: format!("/plugins/{ID}/settings"),
                method: Method::GET,
                route: get().to(api::get_settings),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                }),
            },
            APIRoute {
                path: format!("/plugins/{ID}/settings"),
                method: Method::PUT,
                route: put().to(api::put_settings),
                permission: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                }),
            },
        ]);
        r
    }
}

create_plugin!(Monitor, Monitor::default);
