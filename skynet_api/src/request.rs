use std::{cmp, collections::HashMap, future::Future, hash::Hash, pin::Pin, rc::Rc, sync::Arc};

use actix_cloud::{
    actix_web::{
        self,
        dev::{Payload, ServiceRequest},
        web::Data,
        FromRequest, HttpMessage, HttpRequest,
    },
    async_trait,
    request::Extension,
    router::Checker,
    session::SessionExt,
    utils, Result,
};
use derivative::Derivative;
use enum_as_inner::EnumAsInner;
use sea_orm::{
    sea_query::{self, ConditionExpression, SimpleExpr},
    ColumnTrait, DatabaseConnection, DatabaseTransaction, EntityTrait, FromQueryResult, Order,
    PaginatorTrait, QueryFilter, QueryOrder, Select, TransactionTrait,
};
use serde::{Deserialize, Serialize};
use serde_inline_default::serde_inline_default;
use serde_repr::{Deserialize_repr, Serialize_repr};
use serde_with::{serde_as, DisplayFromStr};
use validator::{Validate, ValidationError};

use crate::{
    permission::{
        PermEntry, PermissionItem, GUEST_ID, GUEST_NAME, PERM_ALL, ROOT_ID, ROOT_NAME, USER_ID,
        USER_NAME,
    },
    HyUuid, Skynet,
};

#[macro_export]
macro_rules! finish {
    ($rsp:expr) => {
        return Ok($rsp)
    };
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

#[derive(Serialize_repr, Deserialize_repr, Debug, PartialEq, Eq, Hash, Clone, Copy)]
#[repr(i32)]
pub enum NotifyLevel {
    Info = 0,
    Success,
    Warning,
    Error,
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

type MenuBadgeFunc = Box<dyn Fn(&Skynet) -> i64 + Send + Sync>;

#[derive(Derivative)]
#[derivative(Default(new = "true"), Debug)]
pub struct MenuItem {
    pub id: HyUuid,
    pub name: String,
    pub path: String,
    pub icon: String,
    pub children: Vec<MenuItem>,
    #[derivative(Debug = "ignore")]
    pub badge_func: Option<MenuBadgeFunc>,
    pub omit_empty: bool,
    pub perm: Option<PermEntry>,
}

impl MenuItem {
    pub fn check(&self, p: &HashMap<HyUuid, PermissionItem>) -> bool {
        self.perm.as_ref().map_or(true, |x| x.check(p))
    }
}

#[derive(Derivative)]
#[derivative(Debug)]
pub struct Request {
    /// user id
    pub uid: Option<HyUuid>,

    /// user name
    pub username: Option<String>,

    /// User permission.
    pub perm: HashMap<HyUuid, PermissionItem>,

    /// actix cloud extension.
    pub extension: Arc<Extension>,
}

impl FromRequest for Request {
    type Error = actix_web::Error;
    type Future = Pin<Box<dyn Future<Output = Result<Self, Self::Error>>>>;

    #[inline]
    fn from_request(req: &HttpRequest, _: &mut Payload) -> Self::Future {
        let req = req.clone();
        Box::pin(async move {
            req.extensions_mut()
                .remove::<Self>()
                .ok_or(actix_web::error::ErrorInternalServerError(
                    "Request is not parsed",
                ))
        })
    }
}

pub type PermChecker = dyn Fn(&HashMap<HyUuid, PermissionItem>) -> bool;

pub enum PermType {
    Entry(PermEntry),
    Custom(Box<PermChecker>),
}

impl From<PermType> for Option<Rc<dyn Checker>> {
    fn from(value: PermType) -> Self {
        Some(Rc::new(AuthChecker::new(value)))
    }
}

pub struct AuthChecker {
    perm: PermType,
}

impl AuthChecker {
    pub fn new(perm: PermType) -> Self {
        Self { perm }
    }

    async fn get_request(req: &mut ServiceRequest) -> Result<Request> {
        let s = req.get_session();
        let id = s.get::<HyUuid>("_id")?;
        let mut perm = match &id {
            Some(x) => {
                let mut perm = if x.is_nil() {
                    // root
                    HashMap::from([(
                        ROOT_ID,
                        PermissionItem {
                            name: ROOT_NAME.to_owned(),
                            pid: ROOT_ID,
                            perm: PERM_ALL,
                            ..Default::default()
                        },
                    )])
                } else {
                    // user
                    let skynet = req.app_data::<Data<Skynet>>().unwrap();
                    let db = req.app_data::<Data<DatabaseConnection>>().unwrap();
                    let tx = db.begin().await.unwrap();
                    let perm = skynet
                        .get_user_perm(&tx, x)
                        .await
                        .unwrap()
                        .into_iter()
                        .map(|x| (x.pid, x))
                        .collect();
                    tx.commit().await.unwrap();
                    perm
                };
                perm.insert(
                    USER_ID,
                    PermissionItem {
                        name: USER_NAME.to_owned(),
                        pid: USER_ID,
                        perm: PERM_ALL,
                        ..Default::default()
                    },
                );
                perm
            }
            None => HashMap::new(),
        };
        perm.insert(
            GUEST_ID,
            PermissionItem {
                name: GUEST_NAME.to_owned(),
                pid: GUEST_ID,
                perm: PERM_ALL,
                ..Default::default()
            },
        );
        Ok(Request {
            uid: id,
            username: s.get::<String>("name")?,
            perm,
            extension: req.extensions().get::<Arc<Extension>>().unwrap().to_owned(),
        })
    }
}

#[async_trait(?Send)]
impl Checker for AuthChecker {
    async fn check(&self, req: &mut ServiceRequest) -> Result<bool> {
        let r = Self::get_request(req).await?;
        let result = match &self.perm {
            PermType::Entry(x) => x.check(&r.perm),
            PermType::Custom(x) => x(&r.perm),
        };
        if result {
            req.extensions_mut().insert(r);
            Ok(true)
        } else {
            Ok(false)
        }
    }
}
