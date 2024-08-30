use actix_web_validator::QsQuery;
use serde::Deserialize;
use skynet_api::{
    actix_cloud::{
        actix_web::{web::Data, Responder},
        response::RspResult,
    },
    entity::notifications::Column,
    request::{
        unique_validator, Condition, IntoExpr, NotifyLevel, PageData, PaginationParam, TimeParam,
    },
    sea_orm::{ColumnTrait, DatabaseConnection, IntoSimpleExpr, TransactionTrait},
    tracing::info,
    Skynet,
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
    param: QsQuery<GetReq>,
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
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
    let data = skynet.notification.find(&tx, cond).await?;
    tx.commit().await?;
    skynet.logger.set_unread(0);
    finish_data!(PageData::new(data));
}

pub async fn delete_all(
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    let cnt = skynet.notification.delete_all(&tx).await?;
    tx.commit().await?;
    info!(success = true, "Delete all notification");
    finish_data!(cnt);
}

pub async fn get_unread(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    finish_data!(skynet.logger.get_unread());
}
