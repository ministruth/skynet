use std::fs;

use actix_cloud::{
    actix_web::web::{Data, Path},
    macros::partial_entity,
    response::{JsonResponse, RspResult},
    state::GlobalState,
    tracing::info,
};
use actix_web_validator::{Json, QsQuery};
use serde::{Deserialize, Serialize};
use skynet_api::{
    entity::{groups, user_histories, users::Column},
    finish,
    permission::{PermEntry, ROOT_ID},
    request::{
        unique_validator, Condition, IDsReq, IntoExpr, PageData, PaginationParam, Request,
        SortType, TimeParam,
    },
    sea_orm::{ColumnTrait, DatabaseConnection, IntoSimpleExpr, TransactionTrait},
    utils::{get_dataurl, parse_dataurl},
    viewer::{groups::GroupViewer, permissions::PermissionViewer, users::UserViewer},
    HyUuid, Skynet,
};
use skynet_macro::common_req;
use validator::Validate;

use crate::{finish_data, finish_err, finish_ok, SkynetResponse};

#[common_req(Column)]
#[derive(Debug, Validate, Deserialize)]
pub struct GetReq {
    pub text: Option<String>,

    pub login_sort: Option<SortType>,
    #[validate(range(min = 0))]
    pub login_start: Option<i64>,
    #[validate(range(min = 0))]
    pub login_end: Option<i64>,

    #[serde(flatten)]
    #[validate(nested)]
    pub page: PaginationParam,
    #[serde(flatten)]
    #[validate(nested)]
    pub time: TimeParam,
}

pub async fn get_all(
    param: QsQuery<GetReq>,
    skynet: Data<Skynet>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    let mut cond = param.common_cond();
    if let Some(text) = &param.text {
        cond = cond.add(
            Condition::any()
                .add(text.like_expr(Column::Id))
                .add(text.like_expr(Column::Username))
                .add(text.like_expr(Column::LastIp)),
        );
    }
    cond = cond.add_option(param.login_start.map(|x| Column::LastLogin.gte(x)));
    cond = cond.add_option(param.login_end.map(|x| Column::LastLogin.lte(x)));
    if let Some(x) = param.login_sort {
        cond = cond.add_sort(Column::LastLogin.into_simple_expr(), x.into());
    };

    let (avatar, mime) = get_dataurl(&fs::read(&skynet.config.avatar)?);
    if mime.is_none() {
        finish_err!(SkynetResponse::UserInvalidAvatar);
    }
    let data = UserViewer::find(db.as_ref(), cond).await?;
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
    finish_data!(PageData::new(data));
}

pub async fn get(
    uid: Path<HyUuid>,
    skynet: Data<Skynet>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    let data = UserViewer::find_by_id(db.as_ref(), &uid).await?;
    if data.is_none() {
        finish!(JsonResponse::not_found());
    }
    let mut data = data.unwrap();
    let (avatar, mime) = if let Some(avatar) = data.avatar {
        get_dataurl(&avatar)
    } else {
        get_dataurl(&fs::read(&skynet.config.avatar)?)
    };
    if mime.is_none() {
        finish_err!(SkynetResponse::UserInvalidAvatar);
    }
    data.avatar = Some(avatar.into());

    finish_data!(data);
}

pub async fn get_self(
    req: Request,
    skynet: Data<Skynet>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    get(req.uid.unwrap().into(), skynet, db).await
}

#[derive(Debug, Validate, Deserialize)]
pub struct AddReq {
    #[validate(length(max = 32))]
    pub username: String,
    pub password: String,
    pub avatar: Option<String>,
    #[validate(custom(function = "unique_validator"))]
    pub group: Option<Vec<HyUuid>>,
    pub base: Option<HyUuid>,
    pub clone_group: Option<bool>,
}

pub async fn add(param: Json<AddReq>, db: Data<DatabaseConnection>) -> RspResult<JsonResponse> {
    if param.clone_group.is_some() && param.base.is_none() {
        finish!(JsonResponse::bad_request(
            "Base should not be None when clone group"
        ));
    }
    if param.base.is_some_and(|x| x.is_nil()) {
        finish_err!(SkynetResponse::UserRoot);
    }
    let avatar = if let Some(x) = &param.avatar {
        let (avatar, mime) = parse_dataurl(x);
        // 1MB
        if avatar.len() > 1024 * 1024 {
            finish!(JsonResponse::bad_request("File too large"));
        }
        if mime.is_none()
            || !["image/png", "image/jpeg", "image/webp"].contains(&mime.unwrap().mime_type())
        {
            finish_err!(SkynetResponse::UserInvalidAvatar);
        }
        Some(avatar)
    } else {
        None
    };
    let tx = db.begin().await?;
    if UserViewer::find_by_name(&tx, &param.username)
        .await?
        .is_some()
    {
        finish_err!(SkynetResponse::UserExist);
    }
    if let Some(group) = &param.group {
        for i in group {
            if GroupViewer::find_by_id(&tx, i).await?.is_none() {
                finish_err!(SkynetResponse::GroupNotExist);
            }
        }
    }
    if let Some(x) = param.base {
        if UserViewer::find_by_id(&tx, &x).await?.is_none() {
            finish_err!(SkynetResponse::UserNotExist);
        }
    }

    let user =
        UserViewer::create(&tx, &param.username, Some(&param.password), avatar, false).await?;
    if let Some(base) = &param.base {
        let perm: Vec<PermEntry> = PermissionViewer::find_user(&tx, base)
            .await?
            .into_iter()
            .map(Into::into)
            .collect();
        PermissionViewer::create_user(&tx, &user.id, &perm).await?;
    }
    let mut group: Vec<HyUuid> = param.group.as_ref().map_or(Vec::new(), ToOwned::to_owned);
    if param.clone_group.is_some_and(|x| x) {
        group.append(
            &mut GroupViewer::find_user_group(&tx, &param.base.unwrap(), false)
                .await?
                .into_iter()
                .map(|x| x.id)
                .collect(),
        );
    }
    if !group.is_empty() {
        group.dedup();
        GroupViewer::link(&tx, &[user.id], &group).await?;
    }
    tx.commit().await?;
    info!(
        success = true,
        username = param.username,
        uid = %user.id,
        gid = ?param.group,
        base = ?param.base,
        clone_group = param.clone_group,
        "Add user",
    );
    finish_data!(user.id);
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutReq {
    #[validate(length(max = 32))]
    pub username: Option<String>,
    pub password: Option<String>,
    pub avatar: Option<String>,
    #[validate(custom(function = "unique_validator"))]
    pub group: Option<Vec<HyUuid>>,
}

pub async fn put(
    uid: Path<HyUuid>,
    param: Json<PutReq>,
    req: Request,
    skynet: Data<Skynet>,
    state: Data<GlobalState>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    if uid.is_nil()
        && (!req.uid.is_some_and(|x| x.is_nil())
            || param.group.as_ref().is_some_and(|x| !x.is_empty()))
    {
        finish_err!(SkynetResponse::UserRoot);
    }

    let tx = db.begin().await?;
    if let Some(user) = UserViewer::find_by_id(&tx, &uid).await? {
        if let Some(name) = &param.username {
            if let Some(x) = UserViewer::find_by_name(&tx, name).await? {
                if x.id != user.id {
                    finish_err!(SkynetResponse::UserExist);
                }
            }
        }
        if let Some(group) = &param.group {
            for i in group {
                if GroupViewer::find_by_id(&tx, i).await?.is_none() {
                    finish_err!(SkynetResponse::GroupNotExist);
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
                    finish!(JsonResponse::bad_request("File too large"));
                }
                if mime.is_none()
                    || !["image/png", "image/jpeg", "image/webp"]
                        .contains(&mime.unwrap().mime_type())
                {
                    finish_err!(SkynetResponse::UserInvalidAvatar);
                }
                Some(avatar)
            }
        } else {
            None
        };

        UserViewer::update(
            &tx,
            state.memorydb.as_ref(),
            &user.id,
            param.username.as_deref(),
            param.password.as_deref(),
            avatar,
            &skynet.config.session.prefix,
        )
        .await?;

        if let Some(gid) = &param.group {
            GroupViewer::unlink(&tx, &[*uid], &[]).await?;
            GroupViewer::link(&tx, &[*uid], gid).await?;
        }
    } else {
        finish!(JsonResponse::not_found());
    }
    tx.commit().await?;
    info!(
        success = true,
        username = param.username,
        uid = %uid,
        gid = ?param.group,
        "Put user",
    );
    finish_ok!();
}

#[derive(Debug, Validate, Deserialize)]
pub struct PutSelfReq {
    pub password: Option<String>,
    pub avatar: Option<String>,
}
pub async fn put_self(
    param: Json<PutSelfReq>,
    req: Request,
    skynet: Data<Skynet>,
    state: Data<GlobalState>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    put(
        req.uid.unwrap().into(),
        Json(PutReq {
            username: None,
            password: param.password.clone(),
            avatar: param.avatar.clone(),
            group: None,
        }),
        req,
        skynet,
        state,
        db,
    )
    .await
}

pub async fn delete_batch(
    param: Json<IDsReq>,
    req: Request,
    skynet: Data<Skynet>,
    state: Data<GlobalState>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    if param.id.contains(&ROOT_ID) && !req.uid.is_some_and(|x| x.is_nil()) {
        finish_err!(SkynetResponse::UserRoot)
    }
    let tx = db.begin().await?;
    let rows = UserViewer::delete(
        &tx,
        state.memorydb.as_ref(),
        &param.id,
        &skynet.config.session.prefix,
    )
    .await?;
    tx.commit().await?;
    if rows != 0 {
        info!(success = true, uid = ?param.id, "Delete users");
    }
    finish_data!(rows);
}

pub async fn delete(
    uid: Path<HyUuid>,
    req: Request,
    skynet: Data<Skynet>,
    state: Data<GlobalState>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    if uid.is_nil() && !req.uid.is_some_and(|x| x.is_nil()) {
        finish_err!(SkynetResponse::UserRoot);
    }
    let tx = db.begin().await?;
    if UserViewer::find_by_id(&tx, &uid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let rows = UserViewer::delete(
        &tx,
        state.memorydb.as_ref(),
        &[*uid],
        &skynet.config.session.prefix,
    )
    .await?;
    tx.commit().await?;
    info!(success = true, uid = %uid, "Delete user");
    finish_data!(rows);
}

pub async fn kick(
    uid: Path<HyUuid>,
    req: Request,
    skynet: Data<Skynet>,
    state: Data<GlobalState>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    if uid.is_nil() && !req.uid.is_some_and(|x| x.is_nil()) {
        finish_err!(SkynetResponse::UserRoot);
    }
    let tx = db.begin().await?;
    if UserViewer::find_by_id(&tx, &uid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    UserViewer::kick(state.memorydb.as_ref(), &uid, &skynet.config.session.prefix).await?;
    tx.commit().await?;
    finish_ok!();
}

pub async fn kick_self(
    req: Request,
    skynet: Data<Skynet>,
    state: Data<GlobalState>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    kick(req.uid.unwrap().into(), req, skynet, state, db).await
}

pub async fn get_group(uid: Path<HyUuid>, db: Data<DatabaseConnection>) -> RspResult<JsonResponse> {
    #[partial_entity(groups::Model)]
    #[derive(Serialize)]
    struct Rsp {
        pub id: HyUuid,
        pub name: String,
        pub created_at: i64,
        pub updated_at: i64,
    }

    let tx = db.begin().await?;
    if UserViewer::find_by_id(&tx, &uid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    let data: Vec<Rsp> = GroupViewer::find_user_group(&tx, &uid, true)
        .await
        .map(|x| (x.into_iter().map(Into::into).collect()))?;
    tx.commit().await?;
    finish_data!(data);
}

#[derive(Debug, Validate, Deserialize)]
pub struct GetHistoryReq {
    pub ip: Option<String>,

    pub time_sort: Option<SortType>,
    #[validate(range(min = 0))]
    pub time_start: Option<i64>,
    #[validate(range(min = 0))]
    pub time_end: Option<i64>,

    #[serde(flatten)]
    #[validate(nested)]
    pub page: PaginationParam,
}

pub async fn get_histories(
    uid: Path<HyUuid>,
    param: QsQuery<GetHistoryReq>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    #[derive(Serialize)]
    struct Rsp {
        id: HyUuid,
        ip: String,
        #[serde(skip_serializing_if = "Option::is_none")]
        user_agent: Option<String>,
        time: i64,
    }
    let tx = db.begin().await?;
    if UserViewer::find_by_id(&tx, &uid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }

    let mut cond = Condition::new(Condition::all()).add_page(param.page.clone());
    cond = cond.add_option(
        param
            .time_start
            .map(|x| user_histories::Column::CreatedAt.gte(x)),
    );
    cond = cond.add_option(
        param
            .time_end
            .map(|x| user_histories::Column::CreatedAt.lte(x)),
    );
    if let Some(x) = param.time_sort {
        cond = cond.add_sort(
            user_histories::Column::CreatedAt.into_simple_expr(),
            x.into(),
        );
    };
    if let Some(ip) = &param.ip {
        cond = cond.add(ip.like_expr(user_histories::Column::Ip));
    }
    let (data, cnt) = UserViewer::find_history_by_id(&tx, &uid, cond).await?;
    let data: Vec<_> = data
        .into_iter()
        .map(|x| Rsp {
            id: x.id,
            ip: x.ip,
            user_agent: x.user_agent,
            time: x.created_at,
        })
        .collect();
    tx.commit().await?;
    finish_data!(PageData::new((data, cnt)));
}

pub async fn get_histories_self(
    param: QsQuery<GetHistoryReq>,
    req: Request,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    get_histories(req.uid.unwrap().into(), param, db).await
}

#[derive(Debug, Validate, Deserialize)]
pub struct GetSessionsReq {
    #[serde(flatten)]
    #[validate(nested)]
    pub page: PaginationParam,
}

pub async fn get_sessions(
    uid: Path<HyUuid>,
    param: QsQuery<GetSessionsReq>,
    skynet: Data<Skynet>,
    state: Data<GlobalState>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    #[derive(Serialize)]
    struct Rsp {
        ttl: u32,
        time: i64,
        #[serde(skip_serializing_if = "Option::is_none")]
        user_agent: Option<String>,
    }
    let tx = db.begin().await?;
    if UserViewer::find_by_id(&tx, &uid).await?.is_none() {
        finish!(JsonResponse::not_found());
    }
    tx.commit().await?;

    let sessions =
        UserViewer::find_sessions(state.memorydb.as_ref(), &uid, &skynet.config.session.prefix)
            .await?;
    let data: Vec<_> = sessions
        .into_iter()
        .map(|x| Rsp {
            ttl: x.ttl,
            time: x.time,
            user_agent: x.user_agent,
        })
        .collect();

    finish_data!(param.page.split(data));
}

pub async fn get_sessions_self(
    param: QsQuery<GetSessionsReq>,
    req: Request,
    skynet: Data<Skynet>,
    state: Data<GlobalState>,
    db: Data<DatabaseConnection>,
) -> RspResult<JsonResponse> {
    get_sessions(req.uid.unwrap().into(), param, skynet, state, db).await
}
