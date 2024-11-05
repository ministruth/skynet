use actix_cloud::Result;
use enum_map::EnumMap;
use migration::Migrator;
use skynet_api::{
    permission::IDTypes::{self, *},
    plugin::init_db,
    sea_orm::{DatabaseConnection, TransactionTrait},
    HyUuid, Skynet,
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

/// # Errors
///
/// Will return `Err` for db error.
pub async fn init(skynet: &Skynet) -> Result<(DatabaseConnection, EnumMap<IDTypes, HyUuid>)> {
    let db = init_db(skynet.config.database.dsn.clone(), Migrator {}).await?;

    let mut ret = EnumMap::<IDTypes, HyUuid>::default();
    // default permission
    let tx = db.begin().await?;
    for (id, note) in default_perm() {
        ret[id] = skynet
            .perm
            .find_or_init(&tx, &id.to_string(), &note)
            .await?
            .id;
    }
    tx.commit().await?;
    Ok((db, ret))
}
