use serde::{Deserialize, Serialize};

#[cfg(all(feature = "plugin-api", feature = "service-skynet"))]
pub static PLUGIN_LOGGER: std::sync::OnceLock<actix_cloud::logger::Logger> =
    std::sync::OnceLock::new();
#[cfg(all(feature = "plugin-api", feature = "service-skynet"))]
pub static PLUGIN_LOGGERGUARD: std::sync::OnceLock<actix_cloud::logger::LoggerGuard> =
    std::sync::OnceLock::new();

#[derive(Serialize, Deserialize, Clone)]
pub struct Logger {
    pub verbose: bool,
    pub json: bool,
    pub enable: bool,
}

#[cfg(all(feature = "plugin-api", feature = "service-skynet"))]
impl Logger {
    pub fn plugin_start(&self, api: crate::service::Service) {
        if self.enable {
            let mut builder = actix_cloud::logger::LoggerBuilder::new().json();
            if self.verbose {
                builder = builder
                    .filename()
                    .line_number()
                    .level(actix_cloud::tracing::Level::DEBUG);
            }
            builder = builder.handler(move |v: &serde_json::Map<String, serde_json::Value>| {
                let api = api.clone();
                let v = serde_json::to_string(v).unwrap();
                Box::pin(async move {
                    api.log(&ffi_rpc::registry::Registry::default(), &v).await;
                    false
                })
            });
            let (logger, guard) = builder.start();
            let _ = PLUGIN_LOGGER.set(logger);
            let _ = PLUGIN_LOGGERGUARD.set(guard);
        }
    }
}
