use actix_cloud::Result;
use enum_map::EnumMap;
use migration::Migrator;
use openssl::{ec, nid};
use sea_orm_migration::MigratorTrait;
use skynet_api::{
    HyUuid,
    config::{
        CONFIG_SESSION_EXPIRE, CONFIG_SESSION_REMEMBER, CONFIG_WEBPUSH_ENDPOINT, CONFIG_WEBPUSH_KEY,
    },
    permission::IDTypes::{self, *},
    sea_orm::{ConnectOptions, Database, DatabaseConnection, TransactionTrait},
    viewer::{permissions::PermissionViewer, settings::SettingViewer},
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
    // setting
    SettingViewer::get_or_init(&tx, CONFIG_SESSION_EXPIRE, "3600").await?;
    SettingViewer::get_or_init(&tx, CONFIG_SESSION_REMEMBER, "5184000").await?;
    // vapid
    if SettingViewer::get(&tx, CONFIG_WEBPUSH_KEY).await?.is_none() {
        let group = ec::EcGroup::from_curve_name(nid::Nid::X9_62_PRIME256V1)?;
        let key = ec::EcKey::generate(&group)?;
        SettingViewer::set_base64(&tx, CONFIG_WEBPUSH_KEY, &key.private_key_to_pem()?).await?;
    }
    // web push
    let endpoint = [
        "android.googleapis.com",
        "fcm.googleapis.com",
        "updates.push.services.mozilla.com",
        "updates-autopush.stage.mozaws.net",
        "updates-autopush.dev.mozaws.net",
        "*.notify.windows.com",
        "*.push.apple.com",
    ];
    SettingViewer::get_or_init(
        &tx,
        CONFIG_WEBPUSH_ENDPOINT,
        &serde_json::to_string(&endpoint)?,
    )
    .await?;
    tx.commit().await?;
    Ok((db, ret))
}
