use skynet::{async_trait, create_plugin, plugin::Plugin, request::APIRoute, Result, Skynet};

#[derive(Debug, Default)]
struct Task;

#[async_trait]
impl Plugin for Task {
    fn on_load(&self, skynet: Skynet) -> (Skynet, Result<()>) {
        (skynet, Ok(()))
    }

    fn on_route(&self, _: &Skynet, r: Vec<APIRoute>) -> Vec<APIRoute> {
        r
    }
}

create_plugin!(Task, Task::default);
