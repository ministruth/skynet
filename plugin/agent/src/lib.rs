use agent_api::{Service, ID, VERSION};
use skynet_api::{
    actix_cloud::state::GlobalState, async_trait, create_plugin, plugin::Plugin, Result, Skynet,
};
use std::{
    path::PathBuf,
    sync::{Arc, OnceLock},
};

static SERVICE: OnceLock<Arc<Service>> = OnceLock::new();

#[derive(Debug, Default)]
struct Agent;

#[async_trait]
impl Plugin for Agent {
    fn on_load(
        &self,
        path: PathBuf,
        mut skynet: Skynet,
        state: GlobalState,
    ) -> (Skynet, GlobalState, Result<()>) {
        let _ = SERVICE.set(Arc::new(Service::new(path)));
        skynet
            .shared_api
            .set(&ID, VERSION, Box::new(SERVICE.get().unwrap().to_owned()));
        (skynet, state, Ok(()))
    }
}

create_plugin!(Agent, Agent::default);
