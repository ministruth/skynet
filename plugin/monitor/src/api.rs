use actix_web_validator::{Json, QsQuery};
use serde::{Deserialize, Serialize};
use serde_json::json;
use skynet::{
    actix_web::{
        web::{Data, Path},
        Responder,
    },
    finish,
    request::{unique_validator, IDsReq, PaginationParam, Request, Response, RspResult},
    sea_orm::TransactionTrait,
    success, HyUuid, Skynet,
};
use validator::Validate;

use crate::{request::ResponseCode, service::AgentStatus, TokenSrv, ID, SERVICE};

pub async fn get_settings(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    #[derive(Serialize)]
    struct Rsp {
        token: String,
    }

    finish!(Response::data(Rsp {
        token: TokenSrv::get(&skynet).unwrap_or_default(),
    }));
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutSettingsReq {
    #[validate(length(max = 32))]
    token: String,
}

pub async fn put_settings(
    param: Json<PutSettingsReq>,
    req: Request,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = srv.db.begin().await?;
    TokenSrv::set(&tx, &skynet, &param.token).await?;
    tx.commit().await?;

    success!(
        "Put monitor settings\n{}",
        json!({
            "ip": req.ip.ip(),
            "plugin": ID,
        })
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

pub async fn get_agents(param: QsQuery<GetAgentsReq>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let data: Vec<serde_json::Value> = srv
        .agent
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
                if !v.id.to_string().contains(x) && !v.name.contains(x) && !v.ip.contains(x) {
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

pub async fn put_agent(
    param: Json<PutAgentsReq>,
    aid: Path<HyUuid>,
    req: Request,
) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = srv.db.begin().await?;
    if srv.agent.find_by_id(&tx, &aid).await?.is_none() {
        finish!(Response::not_found());
    }
    if srv.agent.find_by_name(&tx, &param.name).await?.is_some() {
        finish!(Response::new(ResponseCode::CodeAgentExist));
    }

    srv.agent.rename(&tx, &aid, &param.name).await?;
    tx.commit().await?;

    success!(
        "Put monitor agent\n{}",
        json!({
            "ip": req.ip.ip(),
            "plugin": ID,
            "aid": aid.as_ref(),
            "name": param.name,
        })
    );
    finish!(Response::ok())
}

pub async fn delete_agent(aid: Path<HyUuid>, req: Request) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = srv.db.begin().await?;
    if srv.agent.find_by_id(&tx, &aid).await?.is_none() {
        finish!(Response::not_found());
    }

    let rows = srv.agent.delete(&tx, &aid).await?;
    tx.commit().await?;

    success!(
        "Delete monitor agent\n{}",
        json!({
            "ip": req.ip.ip(),
            "plugin": ID,
            "aid": aid.as_ref(),
        })
    );
    finish!(Response::data(rows))
}

pub async fn delete_agents(param: Json<IDsReq>, req: Request) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let tx = srv.db.begin().await?;
    let mut rows = 0;
    for i in &param.id {
        rows += srv.agent.delete(&tx, i).await?;
    }
    tx.commit().await?;

    success!(
        "Delete monitor agents\n{}",
        json!({
            "ip": req.ip.ip(),
            "plugin": ID,
            "aid": param.id,
        })
    );
    finish!(Response::data(rows))
}
