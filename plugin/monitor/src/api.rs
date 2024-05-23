use actix_web_validator::{Json, QsQuery};
use monitor_service::{AgentStatus, PluginSrv, ID};
use serde::{Deserialize, Serialize};
use serde_json::json;
use skynet::{
    actix_web::{
        web::{Data, Path},
        Responder,
    },
    finish,
    request::{unique_validator, IDsReq, PaginationParam, Response, RspResult},
    sea_orm::TransactionTrait,
    tracing::info,
    HyUuid, Skynet,
};
use skynet_macro::plugin_api;
use validator::Validate;

use crate::{agent_session, request::ResponseCode, AGENT_ADDRESS, DB, RUNTIME, SERVICE};

#[plugin_api(RUNTIME)]
pub async fn get_settings(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    #[allow(clippy::items_after_statements)]
    #[derive(Serialize)]
    struct Rsp {
        token: String,
        shell: Vec<String>,
    }

    finish!(Response::data(Rsp {
        token: PluginSrv::get_token(&skynet).unwrap_or_default(),
        shell: PluginSrv::get_shell_prog(&skynet).unwrap_or_default(),
    }));
}

#[plugin_api(RUNTIME)]
pub async fn get_settings_shell(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    finish!(Response::data(
        PluginSrv::get_shell_prog(&skynet).unwrap_or_default()
    ));
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutSettingsReq {
    #[validate(length(max = 32))]
    token: Option<String>,
    #[validate(custom = "unique_validator")]
    shell: Option<Vec<String>>,
}

#[plugin_api(RUNTIME)]
pub async fn put_settings(
    param: Json<PutSettingsReq>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = DB.get().unwrap().begin().await?;
    if let Some(x) = &param.token {
        PluginSrv::set_token(&tx, &skynet, x).await?;
    }
    if let Some(x) = &param.shell {
        PluginSrv::set_shell_prog(&tx, &skynet, x).await?;
    }
    tx.commit().await?;

    info!(
        success = true,
        plugin = %ID,
        shell = ?param.shell,
        "Put monitor settings",
    );
    finish!(Response::ok())
}

#[derive(Debug, Validate, Deserialize)]
pub struct GetAgentsReq {
    #[validate(custom = "unique_validator")]
    status: Option<Vec<AgentStatus>>,
    text: Option<String>,

    #[serde(flatten)]
    #[validate]
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
    finish!(Response::data(param.page.split(data)));
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutAgentsReq {
    #[validate(length(max = 32))]
    name: String,
}

#[plugin_api(RUNTIME)]
pub async fn put_agent(param: Json<PutAgentsReq>, aid: Path<HyUuid>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = DB.get().unwrap().begin().await?;
    if srv.find_by_id(&tx, &aid).await?.is_none() {
        finish!(Response::not_found());
    }
    if srv.find_by_name(&tx, &param.name).await?.is_some() {
        finish!(Response::new(ResponseCode::CodeAgentExist));
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
    finish!(Response::ok())
}

#[plugin_api(RUNTIME)]
pub async fn reconnect_agent(aid: Path<HyUuid>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = DB.get().unwrap().begin().await?;
    if srv.find_by_id(&tx, &aid).await?.is_none() {
        finish!(Response::not_found());
    }

    if let Some(x) = AGENT_ADDRESS.get().unwrap().read().get(&aid) {
        x.do_send(agent_session::Reconnect);
    }
    tx.commit().await?;
    finish!(Response::ok())
}

#[plugin_api(RUNTIME)]
pub async fn delete_agent(aid: Path<HyUuid>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = DB.get().unwrap().begin().await?;
    if srv.find_by_id(&tx, &aid).await?.is_none() {
        finish!(Response::not_found());
    }

    let rows = srv.delete(&tx, &aid).await?;
    if let Some(x) = AGENT_ADDRESS.get().unwrap().read().get(&aid) {
        x.do_send(agent_session::CloseConnection);
    }
    tx.commit().await?;

    info!(
        success = true,
        plugin = %ID,
        aid = %aid,
        "Delete monitor agent",
    );
    finish!(Response::data(rows))
}

#[plugin_api(RUNTIME)]
pub async fn delete_agents(param: Json<IDsReq>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = DB.get().unwrap().begin().await?;
    let mut rows = 0;
    for i in &param.id {
        rows += srv.delete(&tx, i).await?;
        if let Some(x) = AGENT_ADDRESS.get().unwrap().read().get(i) {
            x.do_send(agent_session::CloseConnection);
        }
    }
    tx.commit().await?;

    info!(
        success = true,
        plugin = %ID,
        aid = ?param.id,
        "Delete monitor agents",
    );
    finish!(Response::data(rows))
}
