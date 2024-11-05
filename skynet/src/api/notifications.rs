use actix_cloud::{
    actix_web::web::Data,
    response::{JsonResponse, RspResult},
    tracing::info,
};
use actix_web_validator::QsQuery;
use serde::Deserialize;
use skynet_api::{
    entity::notifications::Column,
    logger::NotifyLevel,
    request::{
        unique_validator, Condition, IntoExpr, PageData, PaginationParam, Request, TimeParam,
    },
    sea_orm::{ColumnTrait, DatabaseConnection, IntoSimpleExpr, TransactionTrait},
};
use skynet_macro::common_req;
use validator::Validate;

use crate::finish_data;

#[common_req(Column)]
#[derive(Debug, Validate, Deserialize)]
pub struct GetReq {
    #[validate(custom(function = "unique_validator"))]
    pub level: Option<Vec<NotifyLevel>>,
    pub text: Option<String>,

    #[serde(flatten)]
    #[validate(nested)]
    pub page: PaginationParam,
    #[serde(flatten)]
    #[validate(nested)]
    pub time: TimeParam,
}

pub async fn get_all(
    req: Request,
    param: QsQuery<GetReq>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    let mut cond = param.common_cond();
    if let Some(level) = &param.level {
        cond = cond.add(Column::Level.is_in(level.iter().map(|x| *x as i32)));
    }
    if let Some(text) = &param.text {
        cond = cond.add(
            Condition::any()
                .add(text.like_expr(Column::Id))
                .add(text.like_expr(Column::Target))
                .add(text.like_expr(Column::Message))
                .add(text.like_expr(Column::Detail)),
        );
    }
    let tx = db.begin().await?;
    let data = req.skynet.notification.find(&tx, cond).await?;
    tx.commit().await?;
    req.skynet.logger.set_unread(0);
    finish_data!(PageData::new(data));
}

pub async fn delete_all(req: Request, db: Data<DatabaseConnection>) -> RspResult<JsonResponse> {
    let tx = db.begin().await?;
    let cnt = req.skynet.notification.delete_all(&tx).await?;
    tx.commit().await?;
    info!(success = true, "Delete all notification");
    finish_data!(cnt);
}

pub async fn get_unread(req: Request) -> RspResult<JsonResponse> {
    finish_data!(req.skynet.logger.get_unread());
}
