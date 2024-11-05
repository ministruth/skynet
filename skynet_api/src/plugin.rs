use std::path::PathBuf;

use actix_cloud::state::GlobalState;
use anyhow::Result;
use enum_as_inner::EnumAsInner;
use rustls::crypto::aws_lc_rs;
use sea_orm::{ConnectOptions, Database, DatabaseConnection};
use sea_orm_migration::MigratorTrait;
use serde_repr::{Deserialize_repr, Serialize_repr};

use crate::{request::Router, Skynet};

#[derive(
    Serialize_repr, Deserialize_repr, Debug, Clone, Copy, PartialEq, Eq, Hash, EnumAsInner,
)]
#[repr(u8)]
pub enum PluginStatus {
    Unload = 0,
    PendingDisable,
    PendingEnable,
    Enable,
}

/// Plugin interface, all plugins should implement this trait.
///
/// # Lifecycle
///
/// - Skynet init(db, redis, etc.)
/// - Check plugin enabled
/// - **<`_init_logger`>**
/// - **<`on_load`>**
/// - **<`on_route`>**
/// - Skynet running
/// - ...
/// - **<`on_unload`>**
/// - Skynet shutdown
pub trait Plugin: Send + Sync {
    /// Fired to init thread context.
    ///
    /// # Warning
    ///
    /// Do not change this.
    ///
    /// # Errors
    ///
    /// Will return `Err` if context cannot be set.
    fn _init(&self, skynet: &Skynet) -> Result<()> {
        init_rustls();
        skynet.logger.init();
        Ok(())
    }

    /// Fired when the plugin is loaded.
    fn on_load(
        &self,
        _runtime_path: PathBuf,
        skynet: Box<Skynet>,
        state: Box<GlobalState>,
    ) -> (Box<Skynet>, Box<GlobalState>, Result<()>) {
        (skynet, state, Ok(()))
    }

    /// Fired when applying routes.
    fn on_route(&self, _skynet: &Skynet, r: Vec<Router>) -> Vec<Router> {
        r
    }

    /// Fired when the plugin is unloaded.
    fn on_unload(&self, _status: PluginStatus) {}
}

/// Create a plugin.
///
/// # Example
///
/// ```
/// use skynet_api::{create_plugin, plugin::Plugin};
///
/// #[derive(Debug, Default)]
/// struct YourPlugin;
///
/// impl Plugin for YourPlugin {
/// // your implementation
/// }
///
/// create_plugin!(YourPlugin, YourPlugin::default);
/// ```
#[macro_export]
macro_rules! create_plugin {
    ($plugin_type:ty, $constructor:path) => {
        #[no_mangle]
        pub extern "C" fn _plugin_create() -> *mut dyn $crate::plugin::Plugin {
            let constructor: fn() -> $plugin_type = $constructor;
            let boxed: Box<dyn $crate::plugin::Plugin> = Box::new(constructor());
            Box::into_raw(boxed)
        }
    };
}

/// # Errors
///
/// Will return `Err` for db error.
pub async fn init_db<S, M>(dsn: S, _: M) -> Result<DatabaseConnection>
where
    S: Into<String>,
    M: MigratorTrait,
{
    let mut opt = ConnectOptions::new(dsn);
    opt.sqlx_logging(false);
    let db = Database::connect(opt).await?;
    M::up(&db, None).await?;
    Ok(db)
}

pub fn init_rustls() {
    aws_lc_rs::default_provider().install_default().unwrap();
}
