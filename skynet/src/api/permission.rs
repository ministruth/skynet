use std::hash::{Hash, Hasher};

use actix_cloud::{
    actix_web::web::{Data, Path},
    response::{JsonResponse, RspResult},
    tracing::info,
};
use actix_web_validator::Json;
use serde::{Deserialize, Serialize};
use skynet_api::{
    finish,
    permission::UserPerm,
    request::{unique_validator, Condition, Request},
    sea_orm::{DatabaseConnection, TransactionTrait},
    HyUuid,
};
use validator::Validate;

use crate::{finish_data, finish_err, finish_ok, SkynetResponse};

pub async fn get(req: Request, db: Data<DatabaseConnection>) -> RspResult<JsonResponse> {
    let tx = db.begin().await?;
    let data = req.skynet.perm.find(&tx, Condition::default()).await?.0;
    tx.commit().await?;
    finish_data!(data);
}

#[derive(Serialize)]
struct OriginRsp {
    id: HyUuid,
    name: String,
    perm: UserPerm,
}

#[derive(Serialize)]
struct GetRsp {
    #[serde(rename = "id")]
    pid: HyUuid,
    name: String,
    note: String,
    perm: UserPerm,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    origin: Vec<OriginRsp>,
    created_at: i64,
    updated_at: i64,
}

pub async fn get_group(
    req: Request,
    gid: Path<HyUuid>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    let tx = db.begin().await?;
    if req.skynet.group.find_by_id(&tx, &gid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let data: Vec<GetRsp> = req
        .skynet
        .perm
        .find_group(&tx, &gid)
        .await?
        .into_iter()
        .map(|x| GetRsp {
            pid: x.pid,
            name: x.name,
            note: x.note,
            perm: x.perm,
            origin: Vec::new(),
            created_at: x.created_at,
            updated_at: x.updated_at,
        })
        .collect();
    tx.commit().await?;
    finish_data!(data);
}

pub async fn get_user(
    req: Request,
    uid: Path<HyUuid>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    let tx = db.begin().await?;
    if req.skynet.user.find_by_id(&tx, &uid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let data: Vec<GetRsp> = req
        .skynet
        .get_user_perm(&tx, &uid)
        .await?
        .into_iter()
        .map(|x| GetRsp {
            pid: x.pid,
            name: x.name,
            note: x.note,
            perm: x.perm,
            origin: x
                .origin
                .into_iter()
                .map(|x| OriginRsp {
                    id: x.0,
                    name: x.1,
                    perm: x.2,
                })
                .collect(),
            created_at: x.created_at,
            updated_at: x.updated_at,
        })
        .collect();
    tx.commit().await?;
    finish_data!(data);
}

#[derive(Debug, Eq, Validate, Deserialize, Serialize)]
pub struct PutReq {
    pub id: HyUuid,
    #[validate(range(min = -1, max = 7))]
    pub perm: i32,
}

impl PartialEq for PutReq {
    fn eq(&self, other: &Self) -> bool {
        self.id == other.id
    }
}

impl Hash for PutReq {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.id.hash(state);
    }
}

#[derive(Debug, Validate, Deserialize)]
#[serde(transparent)]
pub struct VecPutReq {
    #[validate(nested)]
    #[validate(length(min = 1), custom(function = "unique_validator"))]
    pub inner: Vec<PutReq>,
}

pub async fn put_group(
    req: Request,
    gid: Path<HyUuid>,
    param: Json<VecPutReq>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    let tx = db.begin().await?;
    if req.skynet.group.find_by_id(&tx, &gid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let perm: Vec<HyUuid> = req
        .skynet
        .perm
        .find(&tx, Condition::default())
        .await?
        .0
        .into_iter()
        .map(|x| x.id)
        .collect();
    for i in &param.inner {
        if !perm.contains(&i.id) {
            finish_err!(SkynetResponse::PermissionNotExist);
        }
    }
    for i in &param.inner {
        req.skynet
            .perm
            .grant(&tx, None, Some(&gid), &i.id, i.perm)
            .await?;
    }
    tx.commit().await?;
    info!(
        success = true,
        gid = %gid,
        perm = ?param.inner,
        "Put group permission",
    );
    finish_ok!();
}

pub async fn put_user(
    req: Request,
    uid: Path<HyUuid>,
    param: Json<VecPutReq>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    if uid.is_nil() {
        finish_err!(SkynetResponse::UserRoot);
    }
    let tx = db.begin().await?;
    if req.skynet.user.find_by_id(&tx, &uid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let perm: Vec<HyUuid> = req
        .skynet
        .perm
        .find(&tx, Condition::default())
        .await?
        .0
        .into_iter()
        .map(|x| x.id)
        .collect();
    for i in &param.inner {
        if !perm.contains(&i.id) {
            finish_err!(SkynetResponse::PermissionNotExist);
        }
    }
    for i in &param.inner {
        req.skynet
            .perm
            .grant(&tx, Some(&uid), None, &i.id, i.perm)
            .await?;
    }
    tx.commit().await?;
    info!(success = true, uid = %uid, perm = ?param.inner, "Put user permission");
    finish_ok!();
}
