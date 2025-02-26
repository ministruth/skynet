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

use abi_stable::std_types::RString;
use actix_cloud::{
    config,
    tracing::{debug, error, info},
};
use derivative::Derivative;
use futures::executor::block_on;
use parking_lot::RwLock;
use semver::{Version, VersionReq};
use serde::{Deserialize, Serialize};
use serde_with::{DisplayFromStr, serde_as};
use skynet_api::{
    HyUuid, Result, Skynet, bail,
    ffi_rpc::registry::Registry,
    plugin::{Plugin, PluginStatus},
    request::Router,
    sea_orm::{ConnectionTrait, DatabaseTransaction},
    viewer::settings::SettingViewer,
};
use validator::Validate;
use walkdir::WalkDir;

const PLUGIN_SETTING_PREFIX: &str = "plugin.";
const PLUGIN_CONFIG: &str = "config.yml";

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
    pub instance: Option<Plugin>,
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

pub struct PluginManager {
    plugin: RwLock<Vec<Arc<PluginInstance>>>,
    pub reg: Registry,
}

impl Drop for PluginManager {
    fn drop(&mut self) {
        self.unloadall();
    }
}

impl PluginManager {
    pub fn new(reg: Registry) -> Self {
        Self {
            plugin: Default::default(),
            reg,
        }
    }

    /// Unload all plugins.
    fn unloadall(&mut self) {
        for plugin in self.plugin.write().drain(..) {
            if let Some(x) = &plugin.instance {
                block_on(async {
                    x.on_unload(&self.reg, &plugin.status).await;
                });

                debug!(id = %plugin.id, name = plugin.name, "Plugin unloaded");
            }
        }
    }

    /// Get all instance.
    pub fn get_all(&self) -> Vec<Arc<PluginInstance>> {
        self.plugin.read().clone()
    }

    pub fn delete(&self, id: &HyUuid) {
        self.plugin.write().retain(|v| v.id != *id);
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

    pub fn is_pending(&self) -> bool {
        for i in self.plugin.read().iter() {
            if i.status == PluginStatus::PendingDisable || i.status == PluginStatus::PendingEnable {
                return true;
            }
        }
        false
    }

    /// Set plugin enable status.
    ///
    /// # Errors
    ///
    /// Will return `Err` when db error.
    pub async fn set(
        &self,
        db: &DatabaseTransaction,
        id: &HyUuid,
        enable: bool,
    ) -> Result<Option<Arc<PluginInstance>>> {
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
            SettingViewer::set(db, &setting_name, if enable { "1" } else { "0" }).await?;
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
            Ok(Some(self.plugin.read().index(idx).clone()))
        } else {
            Ok(None)
        }
    }

    /// Load all plugins in folder, ignore error.
    ///
    /// # Panics
    ///
    /// Panics if `dir` cannot be parsed or db error.
    pub async fn load_all<C>(&mut self, db: &C, mut skynet: Skynet, dir: &Path) -> Skynet
    where
        C: ConnectionTrait,
    {
        let mut instance = Self::load_all_internal(&mut self.reg, dir);

        for i in &mut instance {
            if SettingViewer::get(db, &i.setting_name())
                .await
                .unwrap()
                .is_some_and(|x| x == "1")
            {
                i.status = PluginStatus::Enable;
            }
        }

        for i in &mut instance {
            if i.status.is_enable() {
                info!(id = %i.id, name = i.name, "Plugin init");
                let inst = i.instance.as_ref().unwrap();
                match inst
                    .on_load(
                        &self.reg,
                        &skynet,
                        &canonicalize(dir.join(&i.path)).unwrap(),
                    )
                    .await
                {
                    Ok(ret) => {
                        skynet = ret;
                        info!(id = %i.id, name = i.name, "Plugin loaded");
                    }
                    Err(e) => {
                        i.status = PluginStatus::Unload;
                        error!(
                            id = %i.id, name = i.name, error=?e,
                            "Plugin unload because of `on_load` error",
                        )
                    }
                }
            }
            if !i.status.is_enable() {
                i.instance = None;
                self.reg.item.remove(&RString::from(i.id.to_string()));
            }
        }
        self.plugin = RwLock::new(instance.into_iter().map(Arc::new).collect());
        skynet
    }

    /// Parse route.
    pub async fn register(&self, skynet: &Skynet, route: Vec<Router>) -> Vec<Router> {
        let mut route = route;
        for i in self.get_all() {
            if let Some(x) = &i.instance {
                route = x.on_register(&self.reg, skynet, &route).await;
            }
        }
        route
    }

    /// Load plugin .dll/.so/.dylib file.
    fn load_internal<P: AsRef<Path>>(
        reg: &mut Registry,
        config: P,
        filename: P,
    ) -> Result<PluginInstance> {
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

        let api_version = VersionReq::parse(&settings.api_version)?;
        if !api_version.matches(&Version::parse(skynet_api::VERSION).unwrap()) {
            bail!(PluginError::Incompatible(
                settings.name,
                settings.id,
                api_version.to_string()
            ));
        }

        Ok(PluginInstance {
            id: settings.id,
            name: settings.name,
            description: settings.description,
            version: Version::parse(&settings.version)?,
            api_version,
            priority: settings.priority,
            path: path.file_name().unwrap().into(),
            status: PluginStatus::Unload,
            instance: Some(Plugin::new(
                filename.as_ref(),
                reg,
                settings.id.to_string(),
            )?),
        })
    }

    /// Load plugin folder.
    fn load_all_internal<P: AsRef<Path>>(reg: &mut Registry, dir: P) -> Vec<PluginInstance> {
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
                reg,
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
