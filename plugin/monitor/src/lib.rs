use anyhow::Result;
use skynet::create_plugin;
use skynet::plugin::Plugin;
use skynet::request::APIRoute;
use skynet::Skynet;

#[derive(Debug, Default)]
struct Monitor;

impl Plugin for Monitor {
    fn on_load(&self, s: Skynet) -> (Skynet, Result<()>) {
        (s, Ok(()))
    }

    fn on_route(&self, r: Vec<APIRoute>) -> Vec<APIRoute> {
        r
    }
}

create_plugin!(Monitor, Monitor::default);
