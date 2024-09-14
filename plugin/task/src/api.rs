use actix_web_validator::QsQuery;
use serde::{Deserialize, Serialize};
use serde_inline_default::serde_inline_default;
use skynet_api::{
    actix_cloud::{
        actix_web::{web::Path, Responder},
        response::{JsonResponse, RspResult},
    },
    finish,
    request::{Condition, IntoExpr, PageData, PaginationParam, TimeParam},
    sea_orm::{ColumnTrait, IntoSimpleExpr, TransactionTrait},
    tracing::{info, Instrument},
    HyUuid,
};
use skynet_api_task::{entity::tasks, Service, ID};
use skynet_macro::{common_req, plugin_api};
use validator::Validate;

use crate::{TaskResponse, DB, RUNTIME, SERVICE};

#[common_req(tasks::Column)]
#[derive(Debug, Validate, Deserialize)]
pub struct GetTasksReq {
    pub text: Option<String>,

    #[serde(flatten)]
    #[validate(nested)]
    pub page: PaginationParam,
    #[serde(flatten)]
    #[validate(nested)]
    pub time: TimeParam,
}

#[plugin_api(RUNTIME)]
pub async fn get_all(param: QsQuery<GetTasksReq>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    let mut cond = param.common_cond();
    if let Some(text) = &param.text {
        cond = cond.add(
            Condition::any()
                .add(text.like_expr(tasks::Column::Id))
                .add(text.like_expr(tasks::Column::Name))
                .add(text.like_expr(tasks::Column::Detail))
                .add(text.like_expr(tasks::Column::Output)),
        );
    }
    let tx = DB.get().unwrap().begin().await?;
    let data = srv.find(&tx, cond).await?;
    tx.commit().await?;
    finish!(JsonResponse::new(TaskResponse::Success).json(PageData::new(data)));
}

#[serde_inline_default]
#[derive(Debug, Validate, Deserialize)]
pub struct GetOutputReq {
    #[validate(range(min = 0))]
    #[serde_inline_default(0)]
    pub pos: usize,
}

#[plugin_api(RUNTIME)]
pub async fn get_output(
    tid: Path<HyUuid>,
    param: QsQuery<GetOutputReq>,
) -> RspResult<impl Responder> {
    #[derive(Serialize)]
    struct Rsp {
        output: String,
        pos: usize,
    }
    let srv = SERVICE.get().unwrap();
    let tx = DB.get().unwrap().begin().await?;
    let t = match srv.find_by_id(&tx, &tid).await? {
        Some(t) => t.output,
        None => finish!(JsonResponse::not_found()),
    }
    .unwrap_or_default();
    tx.commit().await?;

    let t = if param.pos < t.len() {
        &t[param.pos..]
    } else {
        ""
    };
    finish!(JsonResponse::new(TaskResponse::Success).json(Rsp {
        output: t.to_owned(),
        pos: param.pos + t.len()
    }));
}

#[plugin_api(RUNTIME)]
pub async fn delete_completed() -> RspResult<impl Responder> {
    let tx = DB.get().unwrap().begin().await?;
    let cnt = SERVICE.get().unwrap().delete_completed(&tx).await?;
    tx.commit().await?;
    info!(success = true, plugin = %ID, "Delete all tasks");
    finish!(JsonResponse::new(TaskResponse::Success).json(cnt));
}

#[plugin_api(RUNTIME)]
pub async fn stop(tid: Path<HyUuid>) -> RspResult<impl Responder> {
    let srv = SERVICE.get().unwrap();
    if let Some(tx) = srv.killer_tx.write().remove(&tid) {
        let _ = tx.send(());
    } else {
        finish!(JsonResponse::not_found());
    }

    info!(success = true, plugin = %ID, id = %tid, "Stop task");
    finish!(JsonResponse::new(TaskResponse::Success));
}
