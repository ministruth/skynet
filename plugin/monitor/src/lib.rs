use std::sync::OnceLock;

use sea_orm_migration::{
    sea_query::{Alias, IntoIden},
    MigrationTrait, MigratorTrait,
};
use skynet::{
    async_trait, create_plugin,
    plugin::{self, Plugin},
    request::APIRoute,
    uuid::uuid,
    DatabaseConnection, DynIden, HyUuid, Result, Skynet,
};

mod m20230101_000001_create_table;

static DB: OnceLock<DatabaseConnection> = OnceLock::new();
static ID: HyUuid = HyUuid(uuid!("2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa"));

pub struct Migrator;

#[async_trait]
impl MigratorTrait for Migrator {
    fn migrations() -> Vec<Box<dyn MigrationTrait>> {
        vec![Box::new(m20230101_000001_create_table::Migration)]
    }

    fn migration_table_name() -> DynIden {
        Alias::new(format!("seaql_migrations_{ID}")).into_iden()
    }
}

#[derive(Debug, Default)]
struct Monitor;

#[async_trait]
impl Plugin for Monitor {
    fn on_load(&self, s: Skynet) -> (Skynet, Result<()>) {
        let _ = match plugin::init_db(&s, Migrator {}) {
            Ok(db) => DB.set(db),
            Err(e) => return (s, Err(e)),
        };
        (s, Ok(()))
    }

    fn on_route(&self, r: Vec<APIRoute>) -> Vec<APIRoute> {
        r
    }
}

create_plugin!(Monitor, Monitor::default);
