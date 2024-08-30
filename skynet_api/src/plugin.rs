use std::{
    cmp,
    collections::HashMap,
    env,
    fs::canonicalize,
    ops::{Index, IndexMut},
    path::{Path, PathBuf},
    result,
    sync::Arc,
};

use crate::{HyUuid, Skynet};
use actix_cloud::{
    bail, config,
    router::Router,
    state::GlobalState,
    tracing::{debug, error, info},
    Result,
};
use derivative::Derivative;
use enum_as_inner::EnumAsInner;
use libloading::{Library, Symbol};
use parking_lot::RwLock;
use rustls::crypto::aws_lc_rs;
use sea_orm::{ConnectOptions, Database, DatabaseConnection, DatabaseTransaction};
use sea_orm_migration::MigratorTrait;
use semver::{Version, VersionReq};
use serde::{Deserialize, Serialize};
use serde_repr::{Deserialize_repr, Serialize_repr};
use serde_with::{serde_as, DisplayFromStr};
use validator::Validate;
use walkdir::WalkDir;

const PLUGIN_SETTING_PREFIX: &str = "plugin_";
const PLUGIN_CONFIG: &str = "config.yml";
const PLUGIN_CREATE: &[u8] = b"_plugin_create";

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
        skynet: Skynet,
        state: GlobalState,
    ) -> (Skynet, GlobalState, Result<()>) {
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

#[serde_as]
#[derive(Derivative, Serialize)]
#[derivative(Debug)]
pub struct PluginInstance {
    pub id: HyUuid,
    pub name: String,
    pub description: String,
    #[serde_as(as = "DisplayFromStr")]
    pub version: Version,
    #[serde_as(as = "DisplayFromStr")]
    pub api_version: VersionReq,
    pub priority: i32,
    pub status: PluginStatus,
    #[serde(skip)]
    pub path: PathBuf,

    #[serde(skip)]
    #[derivative(Debug = "ignore")]
    pub instance: Option<Box<dyn Plugin>>,
    #[serde(skip)]
    #[derivative(Debug = "ignore")]
    pub library: Option<Library>,
}

impl Drop for PluginInstance {
    fn drop(&mut self) {
        self.instance = None;
        self.library = None;
    }
}

impl PluginInstance {
    /// Get setting name.
    pub fn setting_name(&self) -> String {
        format!("{}{}", PLUGIN_SETTING_PREFIX, self.id)
    }
}

impl PartialOrd for PluginInstance {
    fn partial_cmp(&self, other: &Self) -> Option<cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for PluginInstance {
    fn cmp(&self, other: &Self) -> cmp::Ordering {
        self.priority.cmp(&other.priority)
    }
}

impl PartialEq for PluginInstance {
    fn eq(&self, other: &Self) -> bool {
        self.id == other.id
    }
}

impl Eq for PluginInstance {}

#[derive(thiserror::Error, Derivative)]
#[derivative(Debug)]
pub enum PluginError {
    #[error("Cannot parse plugin config path")]
    ConfigPath(PathBuf),

    #[error("Plugin `{1}` and `{2}` have conflict id `{0}`")]
    ConflictID(HyUuid, String, String),

    #[error("Plugin `{0}({1})` has incompatible API version {2}")]
    Incompatible(String, HyUuid, String),
}

#[derive(Deserialize, Debug, Validate)]
struct PluginSetting {
    id: HyUuid,
    #[validate(length(min = 1))]
    name: String,
    description: String,
    version: String,
    api_version: String,
    priority: i32,
}

#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct PluginManager {
    plugin: RwLock<Vec<Arc<PluginInstance>>>,
}

impl PluginManager {
    /// Get all instance.
    pub fn get_all(&self) -> &RwLock<Vec<Arc<PluginInstance>>> {
        &self.plugin
    }

    /// Get instance by `id`.
    pub fn get(&self, id: &HyUuid) -> Option<Arc<PluginInstance>> {
        for i in self.plugin.read().iter() {
            if i.id == *id {
                return Some(i.clone());
            }
        }
        None
    }

    /// Set plugin enable status.
    ///
    /// # Errors
    ///
    /// Will return `Err` when db error.
    pub async fn set(
        &self,
        db: &DatabaseTransaction,
        skynet: &Skynet,
        id: &HyUuid,
        enable: bool,
    ) -> Result<bool> {
        let mut idx = None;
        {
            let rlock = self.plugin.read();
            for i in 0..rlock.len() {
                if rlock.index(i).id == *id {
                    idx = Some((i, rlock.index(i).setting_name()));
                    break;
                }
            }
        }
        if let Some((idx, setting_name)) = idx {
            if enable {
                skynet.setting.set(db, &setting_name, "1").await?;
            } else {
                skynet.setting.set(db, &setting_name, "0").await?;
            }
            if let Some(inst) = Arc::get_mut(self.plugin.write().index_mut(idx)) {
                match inst.status {
                    PluginStatus::Unload => {
                        if enable {
                            inst.status = PluginStatus::PendingEnable;
                        }
                    }
                    PluginStatus::PendingDisable => {
                        if enable {
                            inst.status = PluginStatus::Enable;
                        }
                    }
                    PluginStatus::PendingEnable => {
                        if !enable {
                            inst.status = PluginStatus::Unload;
                        }
                    }
                    PluginStatus::Enable => {
                        if !enable {
                            inst.status = PluginStatus::PendingDisable;
                        }
                    }
                }
            }
            Ok(true)
        } else {
            Ok(false)
        }
    }

    /// Load new plugin in `path`.
    ///
    /// Plugin files should be put directly in the `path`. Two files are compulsory:
    /// - `path/*.(dll,dylib,so)`: plugin main file.
    /// - `path/config.yml`: plugin config file.
    ///
    /// Return `Ok(true)` for success load, `Ok(false)` for plugin file not found.
    ///
    /// # Errors
    ///
    /// Will return `Err` when db error.
    pub async fn load(
        &self,
        db: &DatabaseTransaction,
        skynet: &Skynet,
        path: &Path,
    ) -> Result<bool> {
        let mut dll: Vec<PathBuf> = WalkDir::new(path)
            .min_depth(1)
            .max_depth(1)
            .into_iter()
            .filter_entry(|x| {
                x.path()
                    .extension()
                    .is_some_and(|x| x == env::consts::DLL_EXTENSION)
            })
            .map(|x| x.map_or(PathBuf::new(), |x| x.path().to_path_buf()))
            .collect();
        let Some(dll) = dll.pop() else {
            return Ok(false);
        };
        let inst = Self::load_internal(path.join(PLUGIN_CONFIG), dll)?;
        if let Some(x) = self.get(&inst.id) {
            bail!(PluginError::ConflictID(
                x.id,
                inst.name.clone(),
                x.name.clone()
            ));
        }
        skynet.setting.set(db, &inst.setting_name(), "0").await?;
        let mut wlock = self.plugin.write();
        for i in 1..wlock.len() {
            if wlock.index(i).priority > inst.priority {
                wlock.insert(i, Arc::new(inst));
                return Ok(true);
            }
        }
        wlock.push(Arc::new(inst));
        Ok(true)
    }

    /// Load all plugins in folder, ignore error.
    ///
    /// # Panics
    ///
    /// Panics if `dir` cannot be parsed or db error.
    pub fn load_all(
        &mut self,
        skynet: Skynet,
        state: GlobalState,
        dir: &Path,
    ) -> (Skynet, GlobalState) {
        let mut instance = Self::load_all_internal(dir);

        for i in &mut instance {
            if skynet
                .setting
                .get(&i.setting_name())
                .is_some_and(|x| x == "1")
            {
                i.status = PluginStatus::Enable;
            }
        }

        let mut skynet = skynet;
        let mut state = state;
        for i in &mut instance {
            if i.status.is_enable() {
                info!(id = %i.id, name = i.name, "Plugin init");
                let inst = i.instance.as_ref().unwrap();
                if let Err(e) = inst._init(&skynet) {
                    i.status = PluginStatus::Unload;
                    error!(
                        id = %i.id, name = i.name, error=%e,
                        "Plugin unload because of `_init` error",
                    );
                } else {
                    let load_res =
                        inst.on_load(canonicalize(dir.join(&i.path)).unwrap(), skynet, state);
                    skynet = load_res.0;
                    state = load_res.1;
                    if let Err(e) = load_res.2 {
                        i.status = PluginStatus::Unload;
                        error!(
                            id = %i.id, name = i.name, error=%e,
                            "Plugin unload because of `on_load` error",
                        );
                    } else {
                        info!(id = %i.id, name = i.name, "Plugin loaded");
                    }
                }
            }
            if !i.status.is_enable() {
                i.instance = None; // instance MUST be released before library
                i.library = None;
            }
        }
        self.plugin = RwLock::new(instance.into_iter().map(Arc::new).collect());
        (skynet, state)
    }

    /// Parse route.
    pub fn parse_route(&self, skynet: &Skynet, route: Vec<Router>) -> Vec<Router> {
        let mut route = route;
        for i in self.plugin.read().iter() {
            if let Some(x) = &i.instance {
                route = x.on_route(skynet, route);
            }
        }
        route
    }

    /// Load plugin .dll/.so/.dylib file.
    fn load_internal<P: AsRef<Path>>(config: P, filename: P) -> Result<PluginInstance> {
        let config = config
            .as_ref()
            .to_str()
            .ok_or_else(|| PluginError::ConfigPath(config.as_ref().to_path_buf()))?;
        let settings = config::Config::builder()
            .add_source(config::File::with_name(config))
            .build()?;
        let settings: PluginSetting = settings.try_deserialize()?;
        settings.validate()?;
        let mut path = PathBuf::from(filename.as_ref()).canonicalize()?;
        path.pop();
        let mut ret = PluginInstance {
            id: settings.id,
            name: settings.name,
            description: settings.description,
            version: Version::parse(&settings.version)?,
            api_version: VersionReq::parse(&settings.api_version)?,
            priority: settings.priority,
            path: path.file_name().unwrap().into(),
            status: PluginStatus::Unload,
            instance: None,
            library: None,
        };
        if !ret
            .api_version
            .matches(&Version::parse(crate::VERSION).unwrap())
        {
            bail!(PluginError::Incompatible(
                ret.name.clone(),
                ret.id,
                ret.api_version.to_string()
            ));
        }

        // SAFETY: plugin load must be unsafe.
        unsafe {
            type PluginCreate = unsafe fn() -> *mut dyn Plugin;
            let lib = Library::new(filename.as_ref())?;
            let constructor: Symbol<PluginCreate> = lib.get(PLUGIN_CREATE)?;
            let plugin = Box::from_raw(constructor());
            ret.instance = Some(plugin);
            ret.library = Some(lib);

            Ok(ret)
        }
    }

    /// Unload all plugins.
    fn unloadall(&mut self) {
        for plugin in self.plugin.write().drain(..) {
            if let Some(x) = &plugin.instance {
                x.on_unload(plugin.status);
                debug!(id = %plugin.id, name = plugin.name, "Plugin unloaded");
            }
        }
    }

    /// Load plugin folder.
    fn load_all_internal<P: AsRef<Path>>(dir: P) -> Vec<PluginInstance> {
        let mut instance = Vec::new();
        let mut conflict_id: HashMap<HyUuid, String> = HashMap::new();
        for entry in WalkDir::new(dir.as_ref())
            .follow_links(true)
            .min_depth(2)
            .max_depth(2)
            .into_iter()
            .filter_entry(|x| {
                x.path()
                    .extension()
                    .is_some_and(|x| x == env::consts::DLL_EXTENSION)
            })
            .filter_map(result::Result::ok)
        {
            match Self::load_internal(
                entry.path().parent().unwrap().join(PLUGIN_CONFIG),
                entry.path().into(),
            ) {
                Ok(obj) => {
                    if let Some(x) = conflict_id.get(&obj.id) {
                        panic!(
                            "{}",
                            PluginError::ConflictID(obj.id, obj.name.clone(), x.to_owned())
                        );
                    }
                    conflict_id.insert(obj.id, obj.name.clone());
                    instance.push(obj);
                }
                Err(e) => {
                    error!(path=%entry.path().to_string_lossy(), error=%e, "Cannot load plugin")
                }
            }
        }
        instance.sort();
        instance
    }
}

impl Drop for PluginManager {
    fn drop(&mut self) {
        self.unloadall();
    }
}
