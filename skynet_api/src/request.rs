#[macro_export]
macro_rules! finish {
    ($rsp:expr) => {
        return Ok($rsp)
    };
}

#[cfg(feature = "request-pagination")]
mod pagination {
    use std::cmp;

    use anyhow::Result;
    use async_trait::async_trait;
    use derivative::Derivative;
    use sea_orm::{FromQueryResult, prelude::*};
    use serde::{Deserialize, Serialize};
    use serde_inline_default::serde_inline_default;
    use serde_with::{DisplayFromStr, serde_as};
    use validator::Validate;

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
    pub trait SelectPage<E, C>
    where
        E: EntityTrait,
        C: ConnectionTrait,
    {
        async fn select_page(
            self,
            db: &C,
            p: Option<PaginationParam>,
        ) -> Result<(Vec<E::Model>, u64)>;
    }

    #[async_trait]
    impl<M, E, C> SelectPage<E, C> for Select<E>
    where
        E: EntityTrait<Model = M>,
        M: FromQueryResult + Sized + Send + Sync,
        C: ConnectionTrait,
    {
        async fn select_page(
            self,
            db: &C,
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
}

#[cfg(feature = "request-condition")]
mod condition {
    use super::{PaginationParam, SelectPage};

    use anyhow::Result;
    use enum_as_inner::EnumAsInner;
    use sea_orm::{
        FromQueryResult, Order, QueryOrder,
        prelude::*,
        sea_query::{ConditionExpression, LikeExpr, SimpleExpr},
    };
    use serde::{Deserialize, Serialize};

    pub trait IntoExpr {
        fn like_expr<T>(&self, col: T) -> SimpleExpr
        where
            T: ColumnTrait;
    }

    impl IntoExpr for &String {
        fn like_expr<T>(&self, col: T) -> SimpleExpr
        where
            T: ColumnTrait,
        {
            sea_orm::prelude::Expr::col((col.entity_name(), col))
                .like(LikeExpr::new(format!("%{}%", crate::utils::like_escape(self))).escape('\\'))
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

        pub async fn select_page<M, E, C>(
            self,
            q: Select<E>,
            db: &C,
        ) -> Result<(Vec<E::Model>, u64)>
        where
            E: EntityTrait<Model = M>,
            M: FromQueryResult + Sized + Send + Sync,
            C: ConnectionTrait,
        {
            let (q, page) = self.build(q);
            q.select_page(db, page).await
        }
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
}

#[cfg(feature = "request-param")]
mod param {
    use super::*;
    use std::hash::Hash;

    use actix_cloud::utils;
    use anyhow::Result;
    use serde::Deserialize;
    use serde_with::{DisplayFromStr, serde_as};
    use validator::{Validate, ValidationError};

    use crate::HyUuid;

    /// # Errors
    /// Will return `Err` when `x` has duplicate items.
    pub fn unique_validator<T: Eq + Hash>(x: &Vec<T>) -> Result<(), ValidationError> {
        if utils::is_unique(x) {
            Ok(())
        } else {
            Err(ValidationError::new("not unique"))
        }
    }

    #[derive(Debug, Validate, Deserialize)]
    pub struct IDsReq {
        #[validate(length(min = 1, max = 32), custom(function = "unique_validator"))]
        pub id: Vec<HyUuid>,
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
}

#[cfg(feature = "request-route")]
mod route {
    use std::fmt;

    use actix_cloud::{
        actix_web::{Route, http, web},
        router::CSRFType,
    };
    use serde::{Deserialize, Serialize};
    use serde_repr::{Deserialize_repr, Serialize_repr};

    use crate::{HyUuid, permission::PermChecker};

    #[repr(u8)]
    #[derive(Serialize_repr, Deserialize_repr, Clone, Copy)]
    pub enum Method {
        Get,
        Post,
        Put,
        Delete,
        Head,
        Trace,
        Patch,
    }

    impl From<http::Method> for Method {
        fn from(value: http::Method) -> Self {
            match value.as_str() {
                "GET" => Self::Get,
                "POST" => Self::Post,
                "PUT" => Self::Put,
                "DELETE" => Self::Delete,
                "HEAD" => Self::Head,
                "TRACE" => Self::Trace,
                "PATCH" => Self::Patch,
                _ => unimplemented!(),
            }
        }
    }

    impl From<Method> for http::Method {
        fn from(value: Method) -> Self {
            match value {
                Method::Get => http::Method::GET,
                Method::Post => http::Method::POST,
                Method::Put => http::Method::PUT,
                Method::Delete => http::Method::DELETE,
                Method::Head => http::Method::HEAD,
                Method::Trace => http::Method::TRACE,
                Method::Patch => http::Method::PATCH,
            }
        }
    }

    impl Method {
        pub fn get_route(&self) -> Route {
            match self {
                Method::Get => web::get(),
                Method::Post => web::post(),
                Method::Put => web::put(),
                Method::Delete => web::delete(),
                Method::Head => web::head(),
                Method::Trace => web::trace(),
                Method::Patch => web::patch(),
            }
        }

        #[inline]
        pub fn as_str(&self) -> &str {
            match self {
                Method::Get => "GET",
                Method::Post => "POST",
                Method::Put => "PUT",
                Method::Delete => "DELETE",
                Method::Head => "HEAD",
                Method::Trace => "TRACE",
                Method::Patch => "PATCH",
            }
        }
    }

    impl AsRef<str> for Method {
        #[inline]
        fn as_ref(&self) -> &str {
            self.as_str()
        }
    }

    impl fmt::Debug for Method {
        fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
            f.write_str(self.as_ref())
        }
    }

    impl fmt::Display for Method {
        fn fmt(&self, fmt: &mut fmt::Formatter<'_>) -> fmt::Result {
            fmt.write_str(self.as_ref())
        }
    }

    #[derive(Serialize, Deserialize, Clone)]
    pub enum RouterType {
        Inner(String),
        Http(HyUuid, String),
        Websocket(HyUuid, String),
    }

    #[derive(Serialize, Deserialize, Clone)]
    pub struct Router {
        pub path: String,
        pub method: Method,
        pub route: RouterType,
        pub checker: PermChecker,
        pub csrf: CSRFType,
    }
}

#[cfg(feature = "request-req")]
mod req {
    use std::{
        collections::HashMap,
        future::{Ready, ready},
        sync::Arc,
    };

    use actix_cloud::{
        actix_web::{self, FromRequest, HttpMessage, HttpRequest, dev::Payload},
        request::Extension,
    };
    use anyhow::anyhow;
    use serde::{Deserialize, Serialize};

    use crate::{HyUuid, permission::PermissionItem};

    #[derive(Serialize, Deserialize, Debug, Clone)]
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

    impl Request {
        pub fn trace_id(&self) -> HyUuid {
            HyUuid::parse(&self.extension.trace_id).unwrap()
        }
    }

    impl FromRequest for Request {
        type Error = actix_web::Error;
        type Future = Ready<Result<Request, actix_web::Error>>;

        #[inline]
        fn from_request(req: &HttpRequest, _: &mut Payload) -> Self::Future {
            ready(
                req.extensions_mut()
                    .remove::<Self>()
                    .ok_or(actix_web::error::ErrorBadGateway(anyhow!(
                        "Request is not parsed"
                    ))),
            )
        }
    }
}

#[cfg(feature = "request-session")]
mod session {
    use std::{collections::HashMap, str::FromStr};

    use actix_cloud::session;
    use anyhow::Result;

    use crate::HyUuid;

    #[derive(thiserror::Error, Debug)]
    pub enum SessionError {
        #[error("Session field `{0}` is missing")]
        MissingField(String),
    }
    pub struct Session {
        /// Session key.
        pub _key: Option<String>,
        /// Real ttl after refresh.
        pub _ttl: Option<u64>,

        pub id: HyUuid,
        pub name: String,
        pub ttl: u64,
        pub time: i64,
        pub user_agent: Option<String>,
    }

    impl FromStr for Session {
        type Err = anyhow::Error;

        fn from_str(s: &str) -> Result<Self, Self::Err> {
            let v: HashMap<String, String> = serde_json::from_str(s)?;
            let id = serde_json::from_str(
                v.get("_id")
                    .ok_or(SessionError::MissingField(String::from("_id")))?,
            )?;
            let name = serde_json::from_str(
                v.get("name")
                    .ok_or(SessionError::MissingField(String::from("name")))?,
            )?;
            let time = serde_json::from_str(
                v.get("time")
                    .ok_or(SessionError::MissingField(String::from("time")))?,
            )?;
            let ttl = serde_json::from_str(
                v.get("_ttl")
                    .ok_or(SessionError::MissingField(String::from("_ttl")))?,
            )?;
            let user_agent = if let Some(x) = v.get("user_agent") {
                serde_json::from_str(x)?
            } else {
                None
            };
            Ok(Self {
                id,
                name,
                ttl,
                time,
                user_agent,
                _key: None,
                _ttl: None,
            })
        }
    }

    impl Session {
        pub fn into_session(self, s: session::Session) -> Result<()> {
            s.insert("_id", self.id)?;
            s.insert("name", self.name)?;
            s.insert("time", self.time)?;
            s.insert("_ttl", self.ttl)?;
            if let Some(x) = self.user_agent {
                s.insert("user_agent", x)?;
            }
            Ok(())
        }

        pub fn from_session(s: session::Session) -> anyhow::Result<Self> {
            let id = s
                .get("_id")?
                .ok_or(SessionError::MissingField(String::from("_id")))?;
            let name = s
                .get("name")?
                .ok_or(SessionError::MissingField(String::from("name")))?;
            let time = s
                .get("time")?
                .ok_or(SessionError::MissingField(String::from("time")))?;
            let ttl = s
                .get("_ttl")?
                .ok_or(SessionError::MissingField(String::from("_ttl")))?;
            let user_agent = s.get("user_agent")?;
            Ok(Self {
                id,
                name,
                ttl,
                time,
                user_agent,
                _key: None,
                _ttl: None,
            })
        }
    }
}

#[cfg(feature = "request-condition")]
pub use condition::*;
#[cfg(feature = "request-pagination")]
pub use pagination::*;
#[cfg(feature = "request-param")]
pub use param::*;
#[cfg(feature = "request-req")]
pub use req::*;
#[cfg(feature = "request-route")]
pub use route::*;
#[cfg(feature = "request-session")]
pub use session::*;
