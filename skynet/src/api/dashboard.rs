use actix_cloud::{
    actix_web::web::Data,
    response::{JsonResponse, RspResult},
    state::GlobalState,
    t,
    tokio::time::sleep,
};
use serde::Serialize;
use skynet_api::request::Request;
use sysinfo::{CpuRefreshKind, MemoryRefreshKind, RefreshKind, System};

use crate::finish_data;

pub async fn system_info(req: Request, state: Data<GlobalState>) -> RspResult<JsonResponse> {
    #[derive(Serialize)]
    struct Rsp {
        version: String,
        cpu: String,
        memory: u64,
        start_time: i64,
    }
    let sys = System::new_with_specifics(
        RefreshKind::nothing()
            .with_cpu(CpuRefreshKind::everything())
            .with_memory(MemoryRefreshKind::everything()),
    );
    let brand = if sys.cpus().len() > 0 {
        sys.cpus()[0].brand().to_owned()
    } else {
        t!(state.locale, "text.na", &req.extension.lang)
    };
    finish_data!(Rsp {
        version: env!("CARGO_PKG_VERSION").to_owned(),
        cpu: brand,
        memory: sys.total_memory(),
        start_time: state.server.start_time.read().timestamp_millis(),
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
