use std::{cmp, collections::HashMap, future::Future, hash::Hash, pin::Pin, rc::Rc, sync::Arc};

use actix_cloud::{
    actix_web::{
        http::Method,
        web::{Data, Payload},
        FromRequest, HttpRequest, HttpResponse, Route,
    },
    async_trait,
    request::Extension,
    response::{JsonResponse, RspResult},
    router::CSRFType,
    session::Session,
    state::GlobalState,
    utils,
};
use anyhow::Result;
use derivative::Derivative;
use enum_as_inner::EnumAsInner;
use sea_orm::{
    sea_query::{self, ConditionExpression, SimpleExpr},
    ColumnTrait, DatabaseTransaction, EntityTrait, FromQueryResult, Order, PaginatorTrait,
    QueryFilter, QueryOrder, Select,
};
use serde::{Deserialize, Serialize};
use serde_inline_default::serde_inline_default;
use serde_with::{serde_as, DisplayFromStr};
use validator::{Validate, ValidationError};

use crate::{
    permission::{PermType, PermissionItem},
    HyUuid, Skynet,
};

#[macro_export]
macro_rules! finish {
    ($rsp:expr) => {
        return Ok($rsp)
    };
}

macro_rules! impl_router_wrapper {
    ($($x:ident),+) => {
        impl_router_wrapper_json!($($x),+);
        impl_router_wrapper_http!($($x),+);
    };
}

macro_rules! impl_router_wrapper_json {
    ($($x:ident),+) => {
        impl<F, Fut, $($x),+> RouterWrapper<((),$($x,)+), JsonResponse> for F
        where
            F: Fn(Request, $($x),+) -> Fut + 'static,
            Fut: Future<Output = RspResult<JsonResponse>> + 'static,
            $($x: FromRequest,)+
        {
            #[allow(non_snake_case)]
            fn call(self, req: Request, payload: Payload) -> Pin<Box<dyn Future<Output = RspResult<JsonResponse>>>> {
                Box::pin(async move {
                    let mut payload = payload.into_inner();
                    $(
                        let $x = match $x::from_request(&req.http_request, &mut payload).await {
                            Ok(x) => x,
                            Err(e) => return Ok(JsonResponse::bad_request(format!("{}", e.into()))),
                        };
                    )+
                    (self)(req, $($x),+).await
                })
            }
        }
    };
}

macro_rules! impl_router_wrapper_http {
    ($($x:ident),+) => {
        impl<F, Fut, $($x),+> RouterWrapper<((),$($x,)+), HttpResponse> for F
        where
            F: Fn(Request, $($x),+) -> Fut + 'static,
            Fut: Future<Output = RspResult<HttpResponse>> + 'static,
            $($x: FromRequest,)+
        {
            #[allow(non_snake_case)]
            fn call(self, req: Request, payload: Payload) -> Pin<Box<dyn Future<Output = RspResult<HttpResponse>>>> {
                Box::pin(async move {
                    let mut payload = payload.into_inner();
                    $(
                        let $x = match $x::from_request(&req.http_request, &mut payload).await {
                            Ok(x) => x,
                            Err(e) => return Ok(HttpResponse::BadRequest().body(format!("{}", e.into()))),
                        };
                    )+
                    (self)(req, $($x),+).await
                })
            }
        }
    };
}

pub fn box_json_router<T, F>(f: F) -> RouterType
where
    F: RouterWrapper<T, JsonResponse> + Clone + 'static,
{
    RouterType::Json(Rc::new(move |r, p| f.clone().call(r, p)))
}

pub fn box_http_router<T, F>(f: F) -> RouterType
where
    F: RouterWrapper<T, HttpResponse> + Clone + 'static,
{
    RouterType::Http(Rc::new(move |r, p| f.clone().call(r, p)))
}

pub trait RouterWrapper<T, R> {
    fn call(self, req: Request, payload: Payload) -> Pin<Box<dyn Future<Output = RspResult<R>>>>;
}

impl<F, Fut, R> RouterWrapper<(), R> for F
where
    F: Fn() -> Fut + 'static,
    Fut: Future<Output = RspResult<R>> + 'static,
{
    fn call(self, _: Request, _: Payload) -> Pin<Box<dyn Future<Output = RspResult<R>>>> {
        Box::pin((self)())
    }
}

impl<F, Fut, R> RouterWrapper<((),), R> for F
where
    F: Fn(Request) -> Fut + 'static,
    Fut: Future<Output = RspResult<R>> + 'static,
{
    fn call(self, req: Request, _: Payload) -> Pin<Box<dyn Future<Output = RspResult<R>>>> {
        Box::pin((self)(req))
    }
}

impl_router_wrapper!(A1);
impl_router_wrapper!(A1, A2);
impl_router_wrapper!(A1, A2, A3);
impl_router_wrapper!(A1, A2, A3, A4);
impl_router_wrapper!(A1, A2, A3, A4, A5);
impl_router_wrapper!(A1, A2, A3, A4, A5, A6);
impl_router_wrapper!(A1, A2, A3, A4, A5, A6, A7);
impl_router_wrapper!(A1, A2, A3, A4, A5, A6, A7, A8);
impl_router_wrapper!(A1, A2, A3, A4, A5, A6, A7, A8, A9);

pub enum RouterType {
    Json(RouterJsonFunc),
    Http(RouterHttpFunc),
    Raw(Route),
}

pub type RouterJsonFunc =
    Rc<dyn Fn(Request, Payload) -> Pin<Box<dyn Future<Output = RspResult<JsonResponse>>>>>;
pub type RouterHttpFunc =
    Rc<dyn Fn(Request, Payload) -> Pin<Box<dyn Future<Output = RspResult<HttpResponse>>>>>;

pub struct Router {
    pub path: String,
    pub method: Method,
    pub route: RouterType,
    pub checker: PermType,
    pub csrf: CSRFType,
}

pub struct Request {
    /// user id
    pub uid: Option<HyUuid>,

    /// user name
    pub username: Option<String>,

    /// User permission.
    pub perm: HashMap<HyUuid, PermissionItem>,

    /// actix cloud extension.
    pub extension: Arc<Extension>,

    /// Global skynet.
    pub skynet: Data<Skynet>,

    /// actix cloud state.
    pub state: Data<GlobalState>,

    /// session.
    pub session: Session,

    /// original http request.
    pub http_request: HttpRequest,
}

/// # Errors
/// Will return `Err` when `x` has duplicate items.
pub fn unique_validator<T: Eq + Hash>(x: &Vec<T>) -> Result<(), ValidationError> {
    if utils::is_unique(x) {
        Ok(())
    } else {
        Err(ValidationError::new("not unique"))
    }
}

pub trait IntoExpr {
    fn like_expr<T>(&self, col: T) -> sea_query::expr::SimpleExpr
    where
        T: ColumnTrait;
}

impl IntoExpr for &String {
    fn like_expr<T>(&self, col: T) -> sea_query::expr::SimpleExpr
    where
        T: ColumnTrait,
    {
        sea_orm::prelude::Expr::col((col.entity_name(), col)).like(
            sea_query::LikeExpr::new(format!("%{}%", crate::utils::like_escape(self))).escape('\\'),
        )
    }
}

#[derive(Serialize, Derivative)]
#[derivative(Debug)]
pub struct PageData<T> {
    total: u64,
    data: Vec<T>,
}

impl<T> PageData<T> {
    pub fn new(data: (Vec<T>, u64)) -> Self {
        Self {
            total: data.1,
            data: data.0,
        }
    }
}

#[serde_as]
#[serde_inline_default]
#[derive(Debug, Derivative, Deserialize, Validate, Clone)]
#[derivative(Default(new = "true"))]
pub struct PaginationParam {
    #[validate(range(min = 1))]
    #[serde_inline_default(1)]
    #[derivative(Default(value = "1"))]
    #[serde_as(as = "DisplayFromStr")]
    pub page: u64,

    #[validate(range(min = 1, max = 100))]
    #[serde_inline_default(10)]
    #[derivative(Default(value = "10"))]
    #[serde_as(as = "DisplayFromStr")]
    pub size: u64,
}

impl PaginationParam {
    pub fn left(&self) -> u64 {
        self.page.saturating_sub(1).saturating_mul(self.size)
    }

    pub fn right(&self) -> u64 {
        self.page.saturating_mul(self.size)
    }

    fn round(p: u64) -> usize {
        cmp::min(p, usize::MAX as u64) as usize
    }

    pub fn split<T>(&self, data: Vec<T>) -> PageData<T> {
        let cnt = data.len();
        PageData::new((
            data.into_iter()
                .skip(Self::round(self.left()))
                .take(Self::round(self.size))
                .collect(),
            cnt as u64,
        ))
    }
}

#[async_trait]
pub trait SelectPage<E>
where
    E: EntityTrait,
{
    async fn select_page(
        self,
        db: &DatabaseTransaction,
        p: Option<PaginationParam>,
    ) -> Result<(Vec<E::Model>, u64)>;
}

#[async_trait]
impl<M, E> SelectPage<E> for Select<E>
where
    E: EntityTrait<Model = M>,
    M: FromQueryResult + Sized + Send + Sync,
{
    async fn select_page(
        self,
        db: &DatabaseTransaction,
        p: Option<PaginationParam>,
    ) -> Result<(Vec<E::Model>, u64)> {
        if let Some(page) = p {
            let q = self.paginate(db, page.size);
            Ok((q.fetch_page(page.page - 1).await?, q.num_items().await?))
        } else {
            let res = self.all(db).await?;
            let cnt = res.len() as u64;
            Ok((res, cnt))
        }
    }
}

pub struct Condition {
    pub page: Option<PaginationParam>,
    pub cond: sea_orm::Condition,
    pub order: Vec<(SimpleExpr, Order)>,
}

impl Default for Condition {
    fn default() -> Self {
        Self {
            cond: sea_orm::Condition::all(),
            page: None,
            order: Vec::new(),
        }
    }
}

impl From<sea_orm::Condition> for Condition {
    fn from(cond: sea_orm::Condition) -> Self {
        Self::new(cond)
    }
}

impl Condition {
    pub fn new(cond: sea_orm::Condition) -> Self {
        Self {
            cond,
            page: None,
            order: Vec::new(),
        }
    }

    pub fn add_sort(mut self, col: SimpleExpr, sort: Order) -> Self {
        self.order.push((col, sort));
        self
    }

    pub const fn add_page(mut self, page: PaginationParam) -> Self {
        self.page = Some(page);
        self
    }

    pub fn any() -> sea_orm::Condition {
        sea_orm::Condition::any()
    }

    pub fn all() -> sea_orm::Condition {
        sea_orm::Condition::all()
    }

    #[allow(clippy::should_implement_trait)]
    pub fn add<C>(mut self, condition: C) -> Self
    where
        C: Into<ConditionExpression>,
    {
        self.cond = self.cond.add(condition);
        self
    }

    pub fn add_option<C>(mut self, condition: Option<C>) -> Self
    where
        C: Into<ConditionExpression>,
    {
        self.cond = self.cond.add_option(condition);
        self
    }

    pub fn build<E>(self, mut q: Select<E>) -> (Select<E>, Option<PaginationParam>)
    where
        E: EntityTrait,
    {
        for i in self.order {
            q = q.order_by(i.0, i.1);
        }
        (q.filter(self.cond), self.page)
    }

    pub async fn select_page<M, E>(
        self,
        q: Select<E>,
        db: &DatabaseTransaction,
    ) -> Result<(Vec<E::Model>, u64)>
    where
        E: EntityTrait<Model = M>,
        M: FromQueryResult + Sized + Send + Sync,
    {
        let (q, page) = self.build(q);
        q.select_page(db, page).await
    }
}

#[derive(Debug, Validate, Deserialize)]
pub struct IDsReq {
    #[validate(length(min = 1, max = 32), custom(function = "unique_validator"))]
    pub id: Vec<HyUuid>,
}

#[derive(Debug, Serialize, Deserialize, Clone, Copy, EnumAsInner)]
pub enum SortType {
    #[serde(rename = "asc")]
    ASC,
    #[serde(rename = "desc")]
    DESC,
}

impl From<SortType> for Order {
    fn from(val: SortType) -> Self {
        match val {
            SortType::ASC => Self::Asc,
            SortType::DESC => Self::Desc,
        }
    }
}

#[serde_as]
#[derive(Debug, Deserialize, Validate)]
pub struct TimeParam {
    pub created_sort: Option<SortType>,
    #[validate(range(min = 0))]
    #[serde_as(as = "Option<DisplayFromStr>")]
    pub created_start: Option<i64>,
    #[validate(range(min = 0))]
    #[serde_as(as = "Option<DisplayFromStr>")]
    pub created_end: Option<i64>,

    pub updated_sort: Option<SortType>,
    #[validate(range(min = 0))]
    #[serde_as(as = "Option<DisplayFromStr>")]
    pub updated_start: Option<i64>,
    #[validate(range(min = 0))]
    #[serde_as(as = "Option<DisplayFromStr>")]
    pub updated_end: Option<i64>,
}

#[macro_export]
macro_rules! build_time_cond {
    ($cond:ident, $time:expr, $col:expr) => {{
        $cond = $cond.add_option(
            $time
                .created_start
                .map(|x| skynet_api::paste::paste!($col::CreatedAt.gte(x))),
        );
        $cond = $cond.add_option(
            $time
                .created_end
                .map(|x| skynet_api::paste::paste!($col::CreatedAt.lte(x))),
        );
        $cond = $cond.add_option(
            $time
                .updated_start
                .map(|x| skynet_api::paste::paste!($col::UpdatedAt.gte(x))),
        );
        $cond = $cond.add_option(
            $time
                .updated_end
                .map(|x| skynet_api::paste::paste!($col::UpdatedAt.lte(x))),
        );
        if let Some(x) = $time.created_sort {
            $cond = $cond.add_sort(
                skynet_api::paste::paste!($col::CreatedAt.into_simple_expr()),
                x.into(),
            )
        };
        if let Some(x) = $time.updated_sort {
            $cond = $cond.add_sort(
                skynet_api::paste::paste!($col::UpdatedAt.into_simple_expr()),
                x.into(),
            )
        };
        $cond
    }};
}
