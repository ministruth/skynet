use std::sync::atomic::Ordering;

use actix_cloud::{
    actix_web::web::Data,
    response::{JsonResponse, RspResult},
    tracing::info,
};
use actix_web_validator::QsQuery;
use serde::Deserialize;
use skynet_api::{
    entity::notifications::Column,
    request::{unique_validator, Condition, IntoExpr, PageData, PaginationParam, TimeParam},
    sea_orm::{ColumnTrait, DatabaseConnection, IntoSimpleExpr},
    viewer::notifications::NotificationViewer,
};
use skynet_macro::common_req;
use validator::Validate;

use crate::{
    finish_data,
    logger::{self, NotifyLevel},
};

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
    let data = NotificationViewer::find(db.as_ref(), cond).await?;
    logger::UNREAD.store(0, Ordering::SeqCst);
    finish_data!(PageData::new(data));
}

pub async fn delete_all(db: Data<DatabaseConnection>) -> RspResult<JsonResponse> {
    let cnt = NotificationViewer::delete_all(db.as_ref()).await?;
    info!(success = true, "Delete all notification");
    finish_data!(cnt);
}

pub async fn get_unread() -> RspResult<JsonResponse> {
    finish_data!(logger::UNREAD.load(Ordering::Relaxed));
}
