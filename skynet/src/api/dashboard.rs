use actix_cloud::{
    actix_web::web::Data,
    response::{JsonResponse, RspResult},
    state::GlobalState,
    t,
    tokio::time::sleep,
};
use serde::Serialize;
use skynet_api::{Skynet, request::Request};
use sysinfo::{CpuRefreshKind, MemoryRefreshKind, RefreshKind, System};

use crate::finish_data;

pub async fn system_info(
    req: Request,
    state: Data<GlobalState>,
    skynet: Data<Skynet>,
) -> RspResult<JsonResponse> {
    #[derive(Serialize)]
    struct Rsp {
        version: String,
        cpu: String,
        memory: u64,
        start_time: i64,
        warning: Vec<String>,
    }
    let sys = System::new_with_specifics(
        RefreshKind::nothing()
            .with_cpu(CpuRefreshKind::everything())
            .with_memory(MemoryRefreshKind::everything()),
    );
    let brand = if !sys.cpus().is_empty() {
        sys.cpus()[0].brand().to_owned()
    } else {
        t!(state.locale, "text.na", &req.extension.lang)
    };
    let mut warning = Vec::new();
    for i in skynet.warning.iter() {
        warning.push(t!(state.locale, i.value(), &req.extension.lang));
    }
    finish_data!(Rsp {
        version: env!("CARGO_PKG_VERSION").to_owned(),
        cpu: brand,
        memory: sys.total_memory(),
        start_time: state.server.start_time.read().timestamp_millis(),
        warning,
    });
}

pub async fn runtime_info() -> RspResult<JsonResponse> {
    #[derive(Serialize)]
    struct Rsp {
        cpu: f32,
        memory: u64,
        memory_percent: f32,
    }
    let mut sys = System::new_with_specifics(
        RefreshKind::nothing()
            .with_cpu(CpuRefreshKind::everything())
            .with_memory(MemoryRefreshKind::everything()),
    );
    sleep(sysinfo::MINIMUM_CPU_UPDATE_INTERVAL).await;
    sys.refresh_cpu_usage();
    finish_data!(Rsp {
        cpu: sys.global_cpu_usage(),
        memory: sys.used_memory(),
        memory_percent: (sys.used_memory() * 100) as f32 / sys.total_memory() as f32,
    });
}
