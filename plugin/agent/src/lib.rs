use agent_service::{PluginSrv, ID};
use skynet::{async_trait, create_plugin, plugin::Plugin, Result, Skynet};
use std::{
    path::PathBuf,
    sync::{Arc, OnceLock},
};

static SERVICE: OnceLock<Arc<PluginSrv>> = OnceLock::new();

#[derive(Debug, Default)]
struct Agent;

#[async_trait]
impl Plugin for Agent {
    fn on_load(&self, path: PathBuf, mut skynet: Skynet) -> (Skynet, Result<()>) {
        let _ = SERVICE.set(Arc::new(PluginSrv::new(path)));
        skynet
            .shared_api
            .insert(ID, Box::new(SERVICE.get().unwrap().to_owned()));
        (skynet, Ok(()))
    }
}

create_plugin!(Agent, Agent::default);
