use actix_cloud::Result;
use enum_map::EnumMap;
use migration::Migrator;
use sea_orm_migration::MigratorTrait;
use skynet_api::{
    permission::IDTypes::{self, *},
    sea_orm::{ConnectOptions, Database, DatabaseConnection, TransactionTrait},
    viewer::permissions::PermissionViewer,
    HyUuid,
};

fn default_perm() -> Vec<(IDTypes, String)> {
    vec![
        (PermManageUserID, "user management".to_owned()),
        (
            PermManageNotificationID,
            "notification management".to_owned(),
        ),
        (PermManageSystemID, "system management".to_owned()),
        (PermManagePluginID, "plugin management".to_owned()),
    ]
}

pub async fn init(dsn: &str) -> Result<(DatabaseConnection, EnumMap<IDTypes, HyUuid>)> {
    let mut opt = ConnectOptions::new(dsn);
    opt.sqlx_logging(false);
    let db = Database::connect(opt).await?;
    Migrator::up(&db, None).await?;

    let mut ret = EnumMap::<IDTypes, HyUuid>::default();
    let tx = db.begin().await?;
    // default permission
    for (id, note) in default_perm() {
        ret[id] = PermissionViewer::find_or_init(&tx, &id.to_string(), &note)
            .await?
            .id;
    }
    tx.commit().await?;
    Ok((db, ret))
}
