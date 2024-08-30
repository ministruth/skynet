use migration::migrator::Migrator;
use skynet_api::{
    actix_cloud::{
        self,
        actix_web::web::{delete, get, post, put},
        i18n::i18n,
        router::{CSRFType, Router},
        state::GlobalState,
        tokio::runtime::Runtime,
    },
    async_trait, create_plugin,
    parking_lot::RwLock,
    permission::{IDTypes::PermManagePluginID, PermEntry, PERM_ALL, PERM_READ, PERM_WRITE},
    plugin::{self, Plugin},
    request::{MenuItem, PermType},
    sea_orm::{DatabaseConnection, TransactionTrait},
    tracing::{error, info, warn},
    uuid, HyUuid, Result, Skynet,
};
use skynet_api_monitor::{ecies::utils::generate_keypair, Service, ID};
use std::{
    collections::HashMap,
    path::PathBuf,
    sync::{Arc, OnceLock},
};

mod api;
mod migration;
mod server;
mod service;
mod ws;
mod ws_handler;

include!(concat!(env!("OUT_DIR"), "/response.rs"));
static AGENT_API: OnceLock<Arc<skynet_api_agent::Service>> = OnceLock::new();
static SERVICE: OnceLock<Arc<service::Service>> = OnceLock::new();
static RUNTIME: OnceLock<Runtime> = OnceLock::new();
static DB: OnceLock<DatabaseConnection> = OnceLock::new();
static WEB_ADDRESS: OnceLock<RwLock<HashMap<HyUuid, ws::WSAddr>>> = OnceLock::new();

#[derive(Debug, Default)]
struct Monitor;

#[async_trait]
impl Plugin for Monitor {
    fn on_load(
        &self,
        _: PathBuf,
        mut skynet: Skynet,
        mut state: GlobalState,
    ) -> (Skynet, GlobalState, Result<()>) {
        RUNTIME.set(Runtime::new().unwrap()).unwrap();
        WEB_ADDRESS.set(RwLock::default()).unwrap();

        if let Some(api) = skynet
            .shared_api
            .get(&skynet_api_agent::ID, skynet_api_agent::VERSION)
        {
            AGENT_API.set(api).unwrap();
        } else {
            warn!("Agent plugin not enabled, auto update disabled");
        }

        let mut srv = service::Service {
            manage_id: skynet.default_id[PermManagePluginID],
            server: Arc::new(Box::new(server::Server::new())),
            view_id: HyUuid::default(),
            agent: Arc::default(),
        };
        if let Err(e) = RUNTIME.get().unwrap().block_on(async {
            let db = plugin::init_db(&skynet.config.database.dsn, Migrator {}).await?;
            let _ = DB.set(db);

            let tx = DB.get().unwrap().begin().await?;
            if srv.get_setting_address(&skynet).is_none() {
                info!("Addr not found, using default");
                srv.set_setting_address(&tx, &skynet, "0.0.0.0:4242")
                    .await?;
            }
            if srv.get_setting_shell(&skynet).is_none() {
                info!("Shell program not found, using default");
                srv.set_setting_shell(
                    &tx,
                    &skynet,
                    &[
                        String::from("/bin/bash"),
                        String::from("/bin/sh"),
                        String::from("C:\\Windows\\System32\\cmd.exe"),
                    ],
                )
                .await?;
            }
            if srv.get_setting_certificate(&skynet).is_none() {
                info!("Cert not found, generating new one");
                let key = generate_keypair();
                srv.set_setting_certificate(&tx, &skynet, &key.0).await?;
            }
            srv.view_id = skynet
                .perm
                .find_or_init(&tx, &format!("view.plugin.{ID}"), "plugin monitor viewer")
                .await?
                .id;
            srv.init(&tx).await?;
            tx.commit().await?;

            Ok(())
        }) {
            return (skynet, state, Err(e));
        }

        // DB committed. Cannot return err below.
        let _ = skynet.insert_menu(
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
        let _ = skynet.insert_menu(
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
        state.locale = state.locale.add_locale(i18n!("locales"));
        let addr = srv.get_setting_address(&skynet).unwrap();
        let key = srv.get_setting_certificate(&skynet).unwrap();
        let _ = SERVICE.set(Arc::new(srv));
        RUNTIME.get().unwrap().spawn(async move {
            let srv = SERVICE.get().unwrap();
            srv.get_server()
                .start(&addr, key)
                .await
                .map_err(|e| error!(address=addr, error=%e, "Failed to start server"))
        });
        skynet.shared_api.set(
            &ID,
            skynet_api_monitor::VERSION,
            Box::new(SERVICE.get().unwrap().to_owned()),
        );
        (skynet, state, Ok(()))
    }

    fn on_route(&self, skynet: &Skynet, mut r: Vec<Router>) -> Vec<Router> {
        let csrf = CSRFType::Header;
        r.extend(vec![
            Router {
                path: format!("/plugins/{ID}/ws"),
                route: get().to(ws::service),
                checker: PermType::Entry(PermEntry {
                    pid: SERVICE.get().unwrap().view_id,
                    perm: PERM_ALL,
                })
                .into(),
                csrf: CSRFType::ForceParam,
            },
            Router {
                path: format!("/plugins/{ID}/passive_agents"),
                route: get().to(api::get_passive_agents),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/passive_agents"),
                route: post().to(api::add_passive_agents),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/passive_agents"),
                route: delete().to(api::delete_passive_agents_batch),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/passive_agents/{{paid}}"),
                route: delete().to(api::delete_passive_agents),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/agents"),
                route: get().to(api::get_agents),
                checker: PermType::Custom(Box::new(|x| {
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
                }))
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/agents"),
                route: delete().to(api::delete_agents),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/agents/{{aid}}"),
                route: put().to(api::put_agent),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/agents/{{aid}}"),
                route: delete().to(api::delete_agent),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/agents/{{aid}}/reconnect"),
                route: post().to(api::reconnect_agent),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/settings"),
                route: get().to(api::get_settings),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/settings"),
                route: put().to(api::put_settings),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/settings/shell"),
                route: get().to(api::get_settings_shell),
                checker: PermType::Entry(PermEntry {
                    pid: SERVICE.get().unwrap().get_view_id(),
                    perm: PERM_READ,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/settings/certificate"),
                route: get().to(api::get_settings_certificate),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/settings/certificate"),
                route: post().to(api::new_settings_certificate),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/settings/server"),
                route: post().to(api::post_server),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
        ]);
        r
    }
}

create_plugin!(Monitor, Monitor::default);
