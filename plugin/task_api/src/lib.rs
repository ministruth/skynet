use std::{future::Future, pin::Pin};

use entity::tasks;
use skynet_api::{
    actix_cloud::tokio::sync::oneshot, async_trait, request::Condition,
    sea_orm::DatabaseTransaction, uuid, HyUuid, Result,
};

pub mod entity;

pub const VERSION: &str = env!("CARGO_PKG_VERSION");
pub const ID: HyUuid = HyUuid(uuid!("4adaf7d3-b877-43c3-82bd-da3689dc3920"));

#[async_trait]
pub trait TaskItem: Send + Sync {
    async fn update(&self, output: &str, percent: u32) -> Result<()>;
}

#[async_trait]
pub trait Service: Send + Sync {
    async fn create(
        &self,
        name: &str,
        detail: Option<&str>,
        f: impl FnOnce(
                Box<dyn TaskItem>,
                oneshot::Receiver<()>,
            ) -> Pin<Box<dyn Future<Output = Result<i32>> + Send>>
            + Send
            + 'static,
    ) -> Result<HyUuid>;
    async fn find(
        &self,
        db: &DatabaseTransaction,
        cond: Condition,
    ) -> Result<(Vec<tasks::Model>, u64)>;
    async fn find_by_id(
        &self,
        db: &DatabaseTransaction,
        id: &HyUuid,
    ) -> Result<Option<tasks::Model>>;
    async fn delete_completed(&self, db: &DatabaseTransaction) -> Result<u64>;
}
