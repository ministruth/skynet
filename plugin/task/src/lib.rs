use std::{
    collections::HashMap,
    path::PathBuf,
    sync::{Arc, OnceLock},
};

use migration::migrator::Migrator;
use skynet_api::{
    actix_cloud::{
        self,
        actix_web::web::{delete, get, post},
        i18n::i18n,
        router::{CSRFType, Router},
        state::GlobalState,
        tokio::runtime::Runtime,
    },
    async_trait, create_plugin,
    parking_lot::lock_api::RwLock,
    permission::{IDTypes::PermManagePluginID, PermEntry, PERM_READ, PERM_WRITE},
    plugin::{self, Plugin},
    request::{MenuItem, PermType},
    sea_orm::{DatabaseConnection, TransactionTrait},
    uuid, HyUuid, Result, Skynet,
};
use skynet_api_task::ID;

mod api;
mod migration;
mod service;

include!(concat!(env!("OUT_DIR"), "/response.rs"));
static SERVICE: OnceLock<Arc<service::Service>> = OnceLock::new();
static RUNTIME: OnceLock<Runtime> = OnceLock::new();
static DB: OnceLock<DatabaseConnection> = OnceLock::new();

#[derive(Debug, Default)]
struct Task;

#[async_trait]
impl Plugin for Task {
    fn on_load(
        &self,
        _: PathBuf,
        mut skynet: Skynet,
        mut state: GlobalState,
    ) -> (Skynet, GlobalState, Result<()>) {
        RUNTIME.set(Runtime::new().unwrap()).unwrap();
        let srv = service::Service {
            killer_tx: RwLock::new(HashMap::new()),
        };
        if let Err(e) = RUNTIME.get().unwrap().block_on(async {
            let db = plugin::init_db(&skynet.config.database.dsn, Migrator {}).await?;
            let _ = DB.set(db);

            let tx = DB.get().unwrap().begin().await?;
            srv.clean_running(&tx).await?;
            tx.commit().await?;
            Ok(())
        }) {
            return (skynet, state, Err(e));
        }

        let _ = skynet.insert_menu(
            MenuItem {
                id: HyUuid(uuid!("ee689b2e-beaa-43ac-837d-466cad5ff999")),
                name: format!("{ID}.menu.task"),
                path: format!("/plugin/{ID}/"),
                perm: Some(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                }),
                ..Default::default()
            },
            1,
            Some(HyUuid(uuid!("d00d36d0-6068-4447-ab04-f82ce893c04e"))),
        );
        state.locale = state.locale.add_locale(i18n!("locales"));
        let _ = SERVICE.set(Arc::new(srv));
        (skynet, state, Ok(()))
    }

    fn on_route(&self, skynet: &Skynet, mut r: Vec<Router>) -> Vec<Router> {
        let csrf = CSRFType::Header;
        r.extend(vec![
            Router {
                path: format!("/plugins/{ID}/tasks"),
                route: get().to(api::get_all),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/tasks"),
                route: delete().to(api::delete_completed),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_WRITE,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/tasks/{{tid}}/output"),
                route: get().to(api::get_output),
                checker: PermType::Entry(PermEntry {
                    pid: skynet.default_id[PermManagePluginID],
                    perm: PERM_READ,
                })
                .into(),
                csrf,
            },
            Router {
                path: format!("/plugins/{ID}/tasks/{{tid}}/stop"),
                route: post().to(api::stop),
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

create_plugin!(Task, Task::default);
