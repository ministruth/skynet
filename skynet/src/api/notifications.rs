use actix_web::{web::Data, Responder};
use actix_web_validator::QsQuery;
use sea_orm::{ColumnTrait, IntoSimpleExpr, TransactionTrait};
use serde::Deserialize;
use serde_json::json;
use skynet::{
    entity::notifications::Column,
    finish, like_expr,
    request::{
        unique_validator, PageData, PaginationParam, Request, Response, RspResult, TimeParam,
    },
    success, NotifyLevel, Skynet,
};
use skynet_macro::common_req;
use validator::Validate;

#[common_req(Column)]
#[derive(Debug, Validate, Deserialize)]
pub struct GetReq {
    #[validate(custom = "unique_validator")]
    level: Option<Vec<NotifyLevel>>,
    text: Option<String>,

    #[serde(flatten)]
    #[validate]
    page: PaginationParam,
    #[serde(flatten)]
    #[validate]
    time: TimeParam,
}

pub async fn get_all(param: QsQuery<GetReq>, skynet: Data<Skynet>) -> RspResult<impl Responder> {
    let mut cond = param.common_cond();
    if let Some(level) = &param.level {
        cond = cond.add(Column::Level.is_in(level.iter().map(|x| *x as i32)));
    }
    if let Some(text) = &param.text {
        cond = cond.add(
            sea_orm::Condition::any()
                .add(like_expr!(Column::Id, text))
                .add(like_expr!(Column::Target, text))
                .add(like_expr!(Column::Message, text))
                .add(like_expr!(Column::Detail, text)),
        );
    }
    let tx = skynet.db.begin().await?;
    let data = skynet.notification.find(&tx, cond).await?;
    tx.commit().await?;
    skynet.notification.set_unread(0);
    finish!(Response::data(PageData::new(data)));
}

pub async fn delete_all(req: Request, skynet: Data<Skynet>) -> RspResult<impl Responder> {
    let tx = skynet.db.begin().await?;
    let cnt = skynet.notification.delete_all(&tx).await?;
    tx.commit().await?;
    success!(
        "Delete all notification\n{}",
        json!({
            "ip": req.ip.ip(),
        })
    );
    finish!(Response::data(cnt));
}

pub async fn get_unread(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    finish!(Response::data(skynet.notification.get_unread()));
}
