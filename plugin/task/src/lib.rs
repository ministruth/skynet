use std::path::PathBuf;

use skynet_api::{
    actix_cloud::{router::Router, state::GlobalState},
    async_trait, create_plugin,
    plugin::Plugin,
    Result, Skynet,
};

#[derive(Debug, Default)]
struct Task;

#[async_trait]
impl Plugin for Task {
    fn on_load(
        &self,
        _: PathBuf,
        skynet: Skynet,
        state: GlobalState,
    ) -> (Skynet, GlobalState, Result<()>) {
        (skynet, state, Ok(()))
    }

    fn on_route(&self, _: &Skynet, r: Vec<Router>) -> Vec<Router> {
        r
    }
}

create_plugin!(Task, Task::default);
