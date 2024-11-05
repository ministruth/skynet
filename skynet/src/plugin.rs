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

use actix_cloud::{
    config,
    state::GlobalState,
    tracing::{debug, error, info},
};
use derivative::Derivative;
use dlopen2::wrapper::{Container, WrapperApi};
use parking_lot::RwLock;
use semver::{Version, VersionReq};
use serde::{Deserialize, Serialize};
use serde_with::{serde_as, DisplayFromStr};
use skynet_api::{
    bail,
    plugin::{Plugin, PluginStatus},
    request::Router,
    sea_orm::DatabaseTransaction,
    HyUuid, Result, Skynet,
};
use validator::Validate;
use walkdir::WalkDir;

const PLUGIN_SETTING_PREFIX: &str = "plugin_";
const PLUGIN_CONFIG: &str = "config.yml";

#[derive(WrapperApi)]
struct PluginApi {
    _plugin_create: fn() -> *mut dyn Plugin,
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
    library: Option<Container<PluginApi>>,
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

impl Drop for PluginManager {
    fn drop(&mut self) {
        self.unloadall();
    }
}

impl PluginManager {
    /// Unload all plugins.
    fn unloadall(&mut self) {
        for plugin in self.plugin.write().drain(..) {
            if let Some(x) = &plugin.instance {
                x.on_unload(plugin.status);
                debug!(id = %plugin.id, name = plugin.name, "Plugin unloaded");
            }
        }
    }
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

        let mut skynet = Box::new(skynet);
        let mut state = Box::new(state);
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
        (*skynet, *state)
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
            .matches(&Version::parse(skynet_api::VERSION).unwrap())
        {
            bail!(PluginError::Incompatible(
                ret.name.clone(),
                ret.id,
                ret.api_version.to_string()
            ));
        }

        // SAFETY: plugin load must be unsafe.
        unsafe {
            let api: Container<PluginApi> = Container::load(filename.as_ref())?;
            let plugin = Box::from_raw(api._plugin_create());
            ret.instance = Some(plugin);
            ret.library = Some(api);

            Ok(ret)
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
