use std::{sync::Arc, time::Duration};

use actix_web_validator::{Json, QsQuery};
use base64::{engine::general_purpose::STANDARD, Engine};
use serde::{Deserialize, Serialize};
use serde_json::json;
use skynet_api::{
    actix_cloud::{
        actix_web::{
            web::{Data, Path},
            Responder,
        },
        response::{JsonResponse, RspResult},
        tokio::time::sleep,
    },
    finish,
    request::{
        unique_validator, Condition, IDsReq, IntoExpr, PageData, PaginationParam, TimeParam,
    },
    sea_orm::{ColumnTrait, IntoSimpleExpr, TransactionTrait},
    tracing::{error, info, Instrument},
    HyUuid, Skynet,
};
use skynet_api_monitor::{
    ecies::{utils::generate_keypair, PublicKey},
    entity::passive_agents,
    AgentStatus, ReconnectMessage, Service, ID,
};
use skynet_macro::{common_req, plugin_api};
use validator::Validate;

use crate::{service, MonitorResponse, DB, RUNTIME, SERVICE};

#[derive(Debug, Validate, Deserialize)]
pub struct GetAgentsReq {
    #[validate(custom(function = "unique_validator"))]
    status: Option<Vec<AgentStatus>>,
    text: Option<String>,

    #[serde(flatten)]
    #[validate(nested)]
    page: PaginationParam,
}

#[plugin_api(RUNTIME)]
pub async fn get_agents(param: QsQuery<GetAgentsReq>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let data: Vec<serde_json::Value> = srv
        .agent
        .read()
        .iter()
        .filter(|(_, v)| {
            if let Some(x) = &param.status {
                if !x.contains(&v.status) {
                    return false;
                }
            }
            if let Some(x) = &param.text {
                if !v.id.to_string().contains(x)
                    && !v.name.contains(x)
                    && !v.ip.contains(x)
                    && !v.os.as_ref().is_some_and(|v| v.contains(x))
                    && !v.arch.as_ref().is_some_and(|v| v.contains(x))
                {
                    return false;
                }
            }
            true
        })
        .map(|x| json!(x.1))
        .collect();
    finish!(JsonResponse::new(MonitorResponse::Success).json(param.page.split(data)));
}

#[common_req(passive_agents::Column)]
#[derive(Debug, Validate, Deserialize)]
pub struct GetPassiveAgentsReq {
    pub text: Option<String>,

    #[serde(flatten)]
    #[validate(nested)]
    pub page: PaginationParam,
    #[serde(flatten)]
    #[validate(nested)]
    pub time: TimeParam,
}

#[plugin_api(RUNTIME)]
pub async fn get_passive_agents(param: QsQuery<GetPassiveAgentsReq>) -> RspResult<impl Responder> {
    let mut cond = param.common_cond();
    if let Some(text) = &param.text {
        cond = cond.add(
            Condition::any()
                .add(text.like_expr(passive_agents::Column::Id))
                .add(text.like_expr(passive_agents::Column::Name))
                .add(text.like_expr(passive_agents::Column::Address)),
        );
    }
    let tx = DB.get().unwrap().begin().await?;
    let data = SERVICE.get().unwrap().find_passive(&tx, cond).await?;
    tx.commit().await?;

    finish!(JsonResponse::new(MonitorResponse::Success).json(PageData::new(data)));
}

#[derive(Debug, Validate, Deserialize)]
pub struct AddPassiveAgentsReq {
    #[validate(length(min = 1, max = 32))]
    pub name: String,
    #[validate(length(min = 1, max = 64))]
    pub address: String,
    #[validate(range(min = 0))]
    pub retry_time: i32,
}

#[plugin_api(RUNTIME)]
pub async fn add_passive_agents(param: Json<AddPassiveAgentsReq>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = DB.get().unwrap().begin().await?;

    if srv
        .find_passive_by_name(&tx, &param.address)
        .await?
        .is_some()
    {
        finish!(JsonResponse::new(MonitorResponse::PassiveAgentNameExist));
    }
    if srv
        .find_passive_by_address(&tx, &param.address)
        .await?
        .is_some()
    {
        finish!(JsonResponse::new(MonitorResponse::PassiveAgentAddressExist));
    }

    let m = srv
        .create_passive(&tx, &param.name, &param.address, param.retry_time)
        .await?;
    tx.commit().await?;
    srv.server.connect(&m.id);

    info!(
        success = true,
        plugin = %ID,
        name = param.name,
        address = param.address,
        retry_time = param.retry_time,
        "Add passive agent",
    );
    finish!(JsonResponse::new(MonitorResponse::Success).json(m.id));
}

#[plugin_api(RUNTIME)]
pub async fn delete_passive_agents_batch(param: Json<IDsReq>) -> RspResult<impl Responder> {
    let tx = DB.get().unwrap().begin().await?;
    let rows = SERVICE
        .get()
        .unwrap()
        .delete_passive(&tx, &param.id)
        .await?;
    tx.commit().await?;
    if rows != 0 {
        info!(
            success = true,
            plugin = %ID,
            paid = ?param.id,
            "Delete passive agents",
        );
    }
    finish!(JsonResponse::new(MonitorResponse::Success).json(rows));
}

#[plugin_api(RUNTIME)]
pub async fn delete_passive_agents(paid: Path<HyUuid>) -> RspResult<impl Responder> {
    let tx = DB.get().unwrap().begin().await?;
    let rows = SERVICE.get().unwrap().delete_passive(&tx, &[*paid]).await?;
    tx.commit().await?;
    info!(
        success = true,
        plugin = %ID,
        paid = %paid,
        "Delete passive agent",
    );
    finish!(JsonResponse::new(MonitorResponse::Success).json(rows));
}

#[plugin_api(RUNTIME)]
pub async fn get_settings(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    #[derive(Serialize)]
    struct Rsp {
        running: bool,
        shell: Vec<String>,
        address: String,
    }

    let srv = SERVICE.get().unwrap();
    finish!(JsonResponse::new(MonitorResponse::Success).json(Rsp {
        running: srv.get_server().is_running(),
        shell: srv.get_setting_shell(&skynet).unwrap_or_default(),
        address: srv.get_setting_address(&skynet).unwrap_or_default(),
    }));
}

#[plugin_api(RUNTIME)]
pub async fn get_settings_shell(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    finish!(JsonResponse::new(MonitorResponse::Success).json(
        SERVICE
            .get()
            .unwrap()
            .get_setting_shell(&skynet)
            .unwrap_or_default()
    ));
}

#[plugin_api(RUNTIME)]
pub async fn get_settings_certificate(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    let pk = PublicKey::from_secret_key(
        &SERVICE
            .get()
            .unwrap()
            .get_setting_certificate(&skynet)
            .unwrap_or_default(),
    );
    finish!(JsonResponse::file(
        String::from("pubkey"),
        STANDARD.encode(pk.serialize()).into()
    ))
}

async fn restart_server(max_time: u32, srv: Arc<service::Service>, skynet: &Skynet) {
    let addr = srv.get_setting_address(skynet).unwrap_or_default();
    let key = srv.get_setting_certificate(skynet).unwrap_or_default();
    let server = srv.get_server();
    server.stop();
    for _ in 0..max_time {
        if !server.is_running() {
            break;
        }
        sleep(Duration::from_secs(1)).await;
    }
    if !server.is_running() {
        RUNTIME.get().unwrap().spawn(async move {
            srv.get_server()
                .start(&addr, key)
                .await
                .map_err(|e| error!(address=addr, error=%e, "Failed to start server"))
        });
    }
}

#[plugin_api(RUNTIME)]
pub async fn new_settings_certificate(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    let key = generate_keypair();
    let tx = DB.get().unwrap().begin().await?;
    let srv = SERVICE.get().unwrap();
    srv.set_setting_certificate(&tx, &skynet, &key.0).await?;
    tx.commit().await?;

    restart_server(5, srv.to_owned(), &skynet).await;

    info!(
        success = true,
        plugin = %ID,
        "New monitor certificate",
    );
    finish!(JsonResponse::new(MonitorResponse::Success))
}

#[derive(Debug, Validate, Deserialize)]
pub struct PostServerReq {
    pub start: bool,
}

#[plugin_api(RUNTIME)]
pub async fn post_server(
    param: Json<PostServerReq>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let s = srv.get_server();
    if param.start {
        if !s.is_running() {
            let addr = srv.get_setting_address(&skynet).unwrap_or_default();
            let key = srv.get_setting_certificate(&skynet).unwrap_or_default();
            RUNTIME.get().unwrap().spawn(async move {
                srv.get_server()
                    .start(&addr, key)
                    .await
                    .map_err(|e| error!(address=addr, error=%e, "Failed to start server"))
            });
        }
    } else if s.is_running() {
        s.stop();
    }
    info!(
        success = true,
        plugin = %ID,
        start = param.start,
        "Post monitor server",
    );
    finish!(JsonResponse::new(MonitorResponse::Success))
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutSettingsReq {
    #[validate(custom(function = "unique_validator"))]
    pub shell: Option<Vec<String>>,
    pub address: Option<String>,
}

#[plugin_api(RUNTIME)]
pub async fn put_settings(
    param: Json<PutSettingsReq>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = DB.get().unwrap().begin().await?;
    let srv = SERVICE.get().unwrap();
    if let Some(x) = &param.shell {
        srv.set_setting_shell(&tx, &skynet, x).await?;
    }
    if let Some(x) = &param.address {
        srv.set_setting_address(&tx, &skynet, x).await?;
    }
    tx.commit().await?;

    if param.address.is_some() {
        restart_server(5, srv.to_owned(), &skynet).await;
    }

    info!(
        success = true,
        plugin = %ID,
        address = ?param.address,
        shell = ?param.shell,
        "Put monitor settings",
    );
    finish!(JsonResponse::new(MonitorResponse::Success))
}

#[plugin_api(RUNTIME)]
pub async fn reconnect_agent(aid: Path<HyUuid>) -> RspResult<impl Responder> {
    if let Some(agent) = SERVICE.get().unwrap().agent.read().get(&aid) {
        if let Some(x) = &agent.message {
            x.send(skynet_api_monitor::message::Data::Reconnect(
                ReconnectMessage {},
            ))?;
        }
    } else {
        finish!(JsonResponse::not_found());
    }
    info!(
        success = true,
        plugin = %ID,
        aid = %aid,
        "Reconnect monitor agent",
    );
    finish!(JsonResponse::new(MonitorResponse::Success))
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutAgentsReq {
    #[validate(length(max = 32))]
    name: String,
}

#[plugin_api(RUNTIME)]
pub async fn put_agent(param: Json<PutAgentsReq>, aid: Path<HyUuid>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    if !srv.agent_exist(&aid) {
        finish!(JsonResponse::not_found());
    }

    let tx = DB.get().unwrap().begin().await?;
    if srv.find_by_name(&tx, &param.name).await?.is_some() {
        finish!(JsonResponse::new(MonitorResponse::AgentExist));
    }
    srv.rename(&tx, &aid, &param.name).await?;
    tx.commit().await?;

    info!(
        success = true,
        plugin = %ID,
        aid = %aid,
        name = param.name,
        "Put monitor agent",
    );
    finish!(JsonResponse::new(MonitorResponse::Success))
}

#[plugin_api(RUNTIME)]
pub async fn delete_agent(aid: Path<HyUuid>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    if !srv.agent_exist(&aid) {
        finish!(JsonResponse::not_found());
    }

    let tx = DB.get().unwrap().begin().await?;
    let rows = srv.delete(&tx, &aid).await?;
    tx.commit().await?;

    info!(
        success = true,
        plugin = %ID,
        aid = %aid,
        "Delete monitor agent",
    );
    finish!(JsonResponse::new(MonitorResponse::Success).json(rows))
}

#[plugin_api(RUNTIME)]
pub async fn delete_agents(param: Json<IDsReq>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = DB.get().unwrap().begin().await?;
    let mut rows = 0;
    for i in &param.id {
        rows += srv.delete(&tx, i).await?;
    }
    tx.commit().await?;

    info!(
        success = true,
        plugin = %ID,
        aid = ?param.id,
        "Delete monitor agents",
    );
    finish!(JsonResponse::new(MonitorResponse::Success).json(rows))
}
