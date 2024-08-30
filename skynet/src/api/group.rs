use actix_web_validator::{Json, QsQuery};
use serde::{Deserialize, Serialize};
use skynet_api::{
    actix_cloud::{
        actix_web::{
            web::{Data, Path},
            Responder,
        },
        macros::partial_entity,
        response::{JsonResponse, RspResult},
    },
    entity::{groups, users},
    finish,
    hyuuid::uuids2strings,
    permission::PermEntry,
    request::{Condition, IDsReq, IntoExpr, PageData, PaginationParam, TimeParam},
    sea_orm::IntoSimpleExpr,
    sea_orm::{ColumnTrait, DatabaseConnection, TransactionTrait},
    tracing::info,
    HyUuid, Skynet,
};
use skynet_macro::common_req;
use validator::Validate;

use crate::{finish_data, finish_err, finish_ok, SkynetResponse};

#[common_req(groups::Column)]
#[derive(Debug, Validate, Deserialize)]
pub struct GetGroupReq {
    pub text: Option<String>,

    #[serde(flatten)]
    #[validate(nested)]
    pub page: PaginationParam,
    #[serde(flatten)]
    #[validate(nested)]
    pub time: TimeParam,
}

pub async fn get_all(
    param: QsQuery<GetGroupReq>,
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let mut cond = param.common_cond();
    if let Some(text) = &param.text {
        cond = cond.add(
            Condition::any()
                .add(text.like_expr(groups::Column::Id))
                .add(text.like_expr(groups::Column::Name))
                .add(text.like_expr(groups::Column::Note)),
        );
    }
    let tx = db.begin().await?;
    let data = skynet.group.find(&tx, cond).await?;
    tx.commit().await?;
    finish_data!(PageData::new(data));
}

pub async fn get(
    db: Data<DatabaseConnection>,
    gid: Path<HyUuid>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    let data = skynet.group.find_by_id(&tx, &gid).await?;
    if data.is_none() {
        finish!(JsonResponse::not_found());
    }
    tx.commit().await?;
    finish_data!(data);
}

#[common_req(users::Column)]
#[derive(Debug, Validate, Deserialize)]
pub struct GetUserReq {
    pub text: Option<String>,

    #[serde(flatten)]
    #[validate(nested)]
    pub page: PaginationParam,
    #[serde(flatten)]
    #[validate(nested)]
    pub time: TimeParam,
}

pub async fn get_user(
    param: QsQuery<GetUserReq>,
    db: Data<DatabaseConnection>,
    gid: Path<HyUuid>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    #[partial_entity(users::Model)]
    #[derive(Serialize)]
    struct Rsp {
        pub id: HyUuid,
        pub username: String,
        pub created_at: i64,
        pub updated_at: i64,
    }

    let mut cond = param.common_cond();
    if let Some(text) = &param.text {
        cond = cond.add(
            Condition::any()
                .add(text.like_expr(users::Column::Id))
                .add(text.like_expr(users::Column::Username)),
        );
    }
    let tx = db.begin().await?;
    if skynet.group.find_by_id(&tx, &gid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let data: (Vec<Rsp>, u64) = skynet
        .group
        .find_group_user(&tx, &gid, cond)
        .await
        .map(|x| (x.0.into_iter().map(Into::into).collect(), x.1))?;
    tx.commit().await?;
    finish_data!(PageData::new(data));
}

#[derive(Debug, Validate, Deserialize)]
pub struct AddReq {
    #[validate(length(min = 1, max = 32))]
    pub name: String,
    #[validate(length(max = 256))]
    pub note: String,
    pub base: Option<HyUuid>,
    pub clone_user: Option<bool>,
}

pub async fn add(
    param: Json<AddReq>,
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    if param.clone_user.is_some() && param.base.is_none() {
        finish!(JsonResponse::bad_request(
            "Base should not be None when clone user"
        ));
    }
    let tx = db.begin().await?;
    if skynet.group.find_by_name(&tx, &param.name).await?.is_some() {
        finish_err!(SkynetResponse::GroupExist);
    }
    if let Some(x) = param.base {
        if skynet.group.find_by_id(&tx, &x).await?.is_none() {
            finish_err!(SkynetResponse::GroupNotExist);
        }
    }
    let group = skynet.group.create(&tx, &param.name, &param.note).await?;
    if let Some(x) = param.base {
        let perm: Vec<PermEntry> = skynet
            .perm
            .find_group(&tx, &x)
            .await?
            .into_iter()
            .map(Into::into)
            .collect();
        skynet.perm.create_group(&tx, &group.id, &perm).await?;
    }
    if param.clone_user.is_some_and(|x| x) {
        let uid: Vec<HyUuid> = skynet
            .group
            .find_group_user(&tx, &param.base.unwrap(), Condition::default())
            .await?
            .0
            .into_iter()
            .map(|x| x.id)
            .collect();
        skynet.group.link(&tx, &uid, &[group.id]).await?;
    }
    tx.commit().await?;
    info!(
        success = true,
        gid = %group.id,
        base = ?param.base,
        clone_user = param.clone_user,
        name = group.name,
        "Add group"
    );
    finish_data!(group.id);
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutReq {
    #[validate(length(min = 1, max = 32))]
    pub name: Option<String>,
    #[validate(length(max = 256))]
    pub note: Option<String>,
}

pub async fn put(
    gid: Path<HyUuid>,
    param: Json<PutReq>,
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    if let Some(group) = skynet.group.find_by_id(&tx, &gid).await? {
        if let Some(name) = &param.name {
            if let Some(x) = skynet.group.find_by_name(&tx, name).await? {
                if x.id != group.id {
                    finish_err!(SkynetResponse::GroupExist);
                }
            }
        }
        skynet
            .group
            .update(&tx, &group.id, param.name.as_deref(), param.note.as_deref())
            .await?;
    } else {
        finish!(JsonResponse::not_found());
    }
    tx.commit().await?;
    info!(
        success = true,
        gid = %gid,
        "Put group",
    );
    finish_ok!();
}

pub async fn delete_batch(
    param: Json<IDsReq>,
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    let rows = skynet.group.delete(&tx, &param.id).await?;
    tx.commit().await?;
    if rows != 0 {
        info!(
            success = true,
            gid = ?param.id,
            "Delete groups",
        );
    }
    finish_data!(rows);
}

pub async fn delete(
    gid: Path<HyUuid>,
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    if skynet.group.find_by_id(&tx, &gid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let rows = skynet.group.delete(&tx, &[*gid]).await?;
    tx.commit().await?;
    info!(
        success = true,
        gid = %gid,
        "Delete group",
    );
    finish_data!(rows);
}

pub async fn add_user(
    param: Json<IDsReq>,
    db: Data<DatabaseConnection>,
    gid: Path<HyUuid>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    if skynet.group.find_by_id(&tx, &gid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let cnt = skynet
        .user
        .count(
            &tx,
            Condition::new(Condition::all().add(users::Column::Id.is_in(uuids2strings(&param.id)))),
        )
        .await?;
    if cnt != param.id.len() as u64 {
        finish_err!(SkynetResponse::UserNotExist);
    }

    // remove already exist
    let uid: Vec<HyUuid> = skynet
        .group
        .find_group_user(&tx, &gid, Condition::default())
        .await?
        .0
        .into_iter()
        .map(|x| x.id)
        .collect();
    let uid: Vec<HyUuid> = param
        .id
        .iter()
        .filter(|x| !uid.contains(x))
        .map(ToOwned::to_owned)
        .collect();
    if !uid.is_empty() {
        skynet.group.link(&tx, &uid, &[*gid]).await?;
    }
    tx.commit().await?;
    if !uid.is_empty() {
        info!(
            success = true,
            gid = %gid,
            uid=?uid,
            "Add group users",
        );
    }
    finish_ok!();
}

pub async fn delete_user_batch(
    param: Json<IDsReq>,
    db: Data<DatabaseConnection>,
    gid: Path<HyUuid>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    if skynet.group.find_by_id(&tx, &gid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let rows = skynet.group.unlink(&tx, &param.id, &[*gid]).await?;
    tx.commit().await?;
    if rows != 0 {
        info!(
            success = true,
            gid = %gid,
            uid = ?param.id,
            "Delete group users",
        );
    }
    finish_data!(rows);
}

pub async fn delete_user(
    db: Data<DatabaseConnection>,
    ids: Path<(HyUuid, HyUuid)>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let (gid, uid) = ids.into_inner();
    let tx = db.begin().await?;
    if skynet.group.find_by_id(&tx, &gid).await?.is_none()
        || skynet
            .group
            .find_group_user_by_id(&tx, &gid, &uid)
            .await?
            .is_none()
    {
        finish!(JsonResponse::not_found());
    }
    let rows = skynet.group.unlink(&tx, &[uid], &[gid]).await?;
    tx.commit().await?;
    info!(
        success = true,
        gid = %gid,
        uid = %uid,
        "Delete group user",
    );
    finish_data!(rows);
}
