use std::fs;

use actix_web::{
    web::{Data, Path},
    Responder,
};
use actix_web_validator::{Json, QsQuery};
use redis::aio::ConnectionManager;
use sea_orm::{ColumnTrait, DatabaseConnection, IntoSimpleExpr, TransactionTrait};
use serde::{Deserialize, Serialize};
use serde_json::json;
use skynet::{
    entity::{groups, users::Column},
    finish, like_expr,
    permission::PermEntry,
    request::{
        unique_validator, IDsReq, PageData, PaginationParam, Request, Response, ResponseCode,
        RspResult, SortType, TimeParam,
    },
    success,
    utils::{get_dataurl, parse_dataurl},
    HyUuid, Skynet,
};
use skynet_macro::{common_req, partial_entity};
use validator::Validate;

#[common_req(Column)]
#[derive(Debug, Validate, Deserialize)]
pub struct GetReq {
    text: Option<String>,

    login_sort: Option<SortType>,
    #[validate(range(min = 0))]
    login_start: Option<i64>,
    #[validate(range(min = 0))]
    login_end: Option<i64>,

    #[serde(flatten)]
    #[validate]
    page: PaginationParam,
    #[serde(flatten)]
    #[validate]
    time: TimeParam,
}

pub async fn get_all(
    param: QsQuery<GetReq>,
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let mut cond = param.common_cond();
    if let Some(text) = &param.text {
        cond = cond.add(
            sea_orm::Condition::any()
                .add(like_expr!(Column::Id, text))
                .add(like_expr!(Column::Username, text))
                .add(like_expr!(Column::LastIp, text)),
        );
    }
    cond = cond.add_option(param.login_start.map(|x| Column::LastLogin.gte(x)));
    cond = cond.add_option(param.login_start.map(|x| Column::LastLogin.lte(x)));
    if let Some(x) = param.login_sort {
        cond = cond.add_sort(Column::LastLogin.into_simple_expr(), x.into());
    };

    let (avatar, mime) = get_dataurl(&fs::read(skynet.config.avatar.get())?);
    if mime.is_none() {
        finish!(Response::new(ResponseCode::CodeUserInvalidAvatar));
    }
    let tx = db.begin().await?;
    let data = skynet.user.find(&tx, cond).await?;
    let data = (
        data.0
            .into_iter()
            .map(|mut x| {
                x.avatar = Some(
                    x.avatar
                        .map_or_else(|| avatar.clone().into(), |x| get_dataurl(&x).0.into()),
                );
                x
            })
            .collect(),
        data.1,
    );
    tx.commit().await?;
    finish!(Response::data(PageData::new(data)));
}

pub async fn get(
    uid: Path<HyUuid>,
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    let tx = db.begin().await?;
    let data = skynet.user.find_by_id(&tx, &uid).await?;
    if data.is_none() {
        finish!(Response::not_found());
    }
    let mut data = data.unwrap();
    if data.avatar.is_none() {
        let (avatar, mime) = get_dataurl(&fs::read(skynet.config.avatar.get())?);
        if mime.is_none() {
            finish!(Response::new(ResponseCode::CodeUserInvalidAvatar));
        }
        data.avatar = Some(avatar.into());
    }
    tx.commit().await?;
    finish!(Response::data(data));
}

#[derive(Debug, Validate, Deserialize)]
pub struct AddReq {
    #[validate(length(max = 32))]
    username: String,
    password: String,
    avatar: Option<String>,
    #[validate(custom = "unique_validator")]
    group: Option<Vec<HyUuid>>,
    base: Option<HyUuid>,
    clone_group: Option<bool>,
}

pub async fn add(
    param: Json<AddReq>,
    db: Data<DatabaseConnection>,
    req: Request,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    if param.clone_group.is_some() && param.base.is_none() {
        finish!(Response::bad_request(
            "Base should not be None when clone group"
        ));
    }
    if param.username == "root" {
        finish!(Response::new(ResponseCode::CodeUserRoot));
    }
    if param.base.is_some_and(|x| x.is_nil()) {
        finish!(Response::new(ResponseCode::CodeUserRoot));
    }
    let avatar = if let Some(x) = &param.avatar {
        let (avatar, mime) = parse_dataurl(x);
        // 1MB
        if avatar.len() > 1024 * 1024 {
            finish!(Response::bad_request("File too large"));
        }
        if mime.is_none()
            || !["image/png", "image/jpeg", "image/webp"].contains(&mime.unwrap().mime_type())
        {
            finish!(Response::new(ResponseCode::CodeUserInvalidAvatar));
        }
        Some(avatar)
    } else {
        None
    };
    let tx = db.begin().await?;
    if skynet
        .user
        .find_by_name(&tx, &param.username)
        .await?
        .is_some()
    {
        finish!(Response::new(ResponseCode::CodeUserExist));
    }
    if let Some(group) = &param.group {
        for i in group {
            if skynet.group.find_by_id(&tx, i).await?.is_none() {
                finish!(Response::new(ResponseCode::CodeGroupNotExist));
            }
        }
    }
    if let Some(x) = param.base {
        if skynet.user.find_by_id(&tx, &x).await?.is_none() {
            finish!(Response::new(ResponseCode::CodeUserNotExist));
        }
    }

    let user = skynet
        .user
        .create(&tx, &skynet, &param.username, Some(&param.password), avatar)
        .await?;
    if let Some(base) = &param.base {
        let perm: Vec<PermEntry> = skynet
            .perm
            .find_user(&tx, base)
            .await?
            .into_iter()
            .map(Into::into)
            .collect();
        skynet.perm.create_user(&tx, &user.id, &perm).await?;
    }
    let mut group: Vec<HyUuid> = param.group.as_ref().map_or(Vec::new(), ToOwned::to_owned);
    if param.clone_group.is_some_and(|x| x) {
        group.append(
            &mut skynet
                .group
                .find_user_group(&tx, &param.base.unwrap(), false)
                .await?
                .into_iter()
                .map(|x| x.id)
                .collect(),
        );
    }
    if !group.is_empty() {
        group.dedup();
        skynet.group.link(&tx, &[user.id], &group).await?;
    }
    tx.commit().await?;
    success!(
        "Add user\n{}",
        json!({
            "username": param.username,
            "uid": user.id,
            "gid": param.group,
            "base": param.base,
            "clone_group": param.clone_group,
            "ip": req.ip.ip(),
        })
    );
    finish!(Response::data(user.id));
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutReq {
    #[validate(length(max = 32))]
    username: Option<String>,
    password: Option<String>,
    avatar: Option<String>,
    #[validate(custom = "unique_validator")]
    group: Option<Vec<HyUuid>>,
}

pub async fn put(
    uid: Path<HyUuid>,
    param: Json<PutReq>,
    db: Data<DatabaseConnection>,
    redis: Data<ConnectionManager>,
    req: Request,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    if uid.is_nil()
        && (!req.uid.is_some_and(|x| x.is_nil())
            || param.username.as_ref().is_some_and(|x| x != "root")
            || param.group.as_ref().is_some_and(|x| !x.is_empty()))
    {
        finish!(Response::new(ResponseCode::CodeUserRoot));
    }

    if param.username.as_ref().is_some_and(|x| x == "root") {
        finish!(Response::new(ResponseCode::CodeUserRoot));
    }
    let tx = db.begin().await?;
    if let Some(user) = skynet.user.find_by_id(&tx, &uid).await? {
        if let Some(name) = &param.username {
            if let Some(x) = skynet.user.find_by_name(&tx, name).await? {
                if x.id != user.id {
                    finish!(Response::new(ResponseCode::CodeUserExist));
                }
            }
        }
        if let Some(group) = &param.group {
            for i in group {
                if skynet.group.find_by_id(&tx, i).await?.is_none() {
                    finish!(Response::new(ResponseCode::CodeGroupNotExist));
                }
            }
        }
        let avatar = if let Some(x) = &param.avatar {
            if x.is_empty() {
                Some(Vec::new())
            } else {
                let (avatar, mime) = parse_dataurl(x);
                // 1MB
                if avatar.len() > 1024 * 1024 {
                    finish!(Response::bad_request("File too large"));
                }
                if mime.is_none()
                    || !["image/png", "image/jpeg", "image/webp"]
                        .contains(&mime.unwrap().mime_type())
                {
                    finish!(Response::new(ResponseCode::CodeUserInvalidAvatar));
                }
                Some(avatar)
            }
        } else {
            None
        };

        skynet
            .user
            .update(
                &tx,
                &redis,
                &skynet,
                &user.id,
                param.username.as_deref(),
                param.password.as_deref(),
                avatar,
                &user.salt_prefix,
                &user.salt_suffix,
            )
            .await?;

        if let Some(gid) = &param.group {
            skynet.group.unlink(&tx, &[*uid], &[]).await?;
            skynet.group.link(&tx, &[*uid], gid).await?;
        }
    } else {
        finish!(Response::not_found());
    }
    tx.commit().await?;
    success!(
        "Put user\n{}",
        json!({
            "username": param.username,
            "uid": uid.as_ref(),
            "gid": param.group,
            "ip": req.ip.ip(),
        })
    );
    finish!(Response::ok());
}

type DeleteBatchReq = IDsReq;

pub async fn delete_batch(
    param: Json<DeleteBatchReq>,
    db: Data<DatabaseConnection>,
    redis: Data<ConnectionManager>,
    req: Request,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    if param.id.contains(&HyUuid::nil()) && !req.uid.is_some_and(|x| x.is_nil()) {
        finish!(Response::new(ResponseCode::CodeUserRoot));
    }
    let tx = db.begin().await?;
    let rows = skynet.user.delete(&tx, &redis, &skynet, &param.id).await?;
    tx.commit().await?;
    if rows != 0 {
        success!(
            "Delete users\n{}",
            json!({
                "uid": param.id,
                "ip": req.ip.ip(),
            })
        );
    }
    finish!(Response::data(rows));
}

pub async fn delete(
    uid: Path<HyUuid>,
    db: Data<DatabaseConnection>,
    redis: Data<ConnectionManager>,
    req: Request,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    if uid.is_nil() && !req.uid.is_some_and(|x| x.is_nil()) {
        finish!(Response::new(ResponseCode::CodeUserRoot));
    }
    let tx = db.begin().await?;
    if skynet.user.find_by_id(&tx, &uid).await?.is_none() {
        finish!(Response::not_found());
    }
    let rows = skynet.user.delete(&tx, &redis, &skynet, &[*uid]).await?;
    tx.commit().await?;
    success!(
        "Delete user\n{}",
        json!({
            "uid": uid.as_ref(),
            "ip": req.ip.ip(),
        })
    );
    finish!(Response::data(rows));
}

pub async fn kick(
    uid: Path<HyUuid>,
    db: Data<DatabaseConnection>,
    redis: Data<ConnectionManager>,
    req: Request,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    if uid.is_nil() && !req.uid.is_some_and(|x| x.is_nil()) {
        finish!(Response::new(ResponseCode::CodeUserRoot));
    }
    let tx = db.begin().await?;
    if skynet.user.find_by_id(&tx, &uid).await?.is_none() {
        finish!(Response::not_found());
    }
    skynet.user.kick(&redis, &skynet, &uid).await?;
    tx.commit().await?;
    finish!(Response::ok());
}

pub async fn get_group(
    uid: Path<HyUuid>,
    db: Data<DatabaseConnection>,
    skynet: Data<Skynet>,
) -> RspResult<impl Responder> {
    #[partial_entity(groups::Model)]
    #[derive(Serialize)]
    struct Rsp {
        pub id: HyUuid,
        pub name: String,
        pub created_at: i64,
        pub updated_at: i64,
    }

    let tx = db.begin().await?;
    if skynet.user.find_by_id(&tx, &uid).await?.is_none() {
        finish!(Response::not_found());
    }
    let data: Vec<Rsp> = skynet
        .group
        .find_user_group(&tx, &uid, true)
        .await
        .map(|x| (x.into_iter().map(Into::into).collect()))?;
    tx.commit().await?;
    finish!(Response::data(data));
}
