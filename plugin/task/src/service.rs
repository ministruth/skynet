use std::collections::HashMap;
use std::future::Future;
use std::pin::Pin;

use sea_orm_migration::prelude::Expr;
use skynet_api::actix_cloud::tokio::spawn;
use skynet_api::actix_cloud::tokio::sync::oneshot;
use skynet_api::actix_cloud::Error;
use skynet_api::parking_lot::RwLock;
use skynet_api::request::Condition;
use skynet_api::sea_orm::{
    ActiveModelTrait, ColumnTrait, DatabaseTransaction, EntityTrait, QueryFilter, Set,
    TransactionTrait,
};
use skynet_api::tracing::error;
use skynet_api::{anyhow, async_trait, HyUuid, Result};
use skynet_api_task::entity::tasks;

use crate::DB;

struct TaskItem {
    id: HyUuid,
}

impl TaskItem {
    fn new(id: &HyUuid) -> Self {
        Self { id: id.to_owned() }
    }
}

#[async_trait]
impl skynet_api_task::TaskItem for TaskItem {
    async fn update(&self, output: &str, percent: u32) -> Result<()> {
        let tx = DB.get().unwrap().begin().await?;
        let mut m = tasks::Entity::find_by_id(self.id)
            .one(&tx)
            .await?
            .ok_or(anyhow!("Task not found"))?;
        let output = m.output.take().unwrap_or_default() + output;
        let percent = m.percent.saturating_add(percent.try_into()?);
        let mut m: tasks::ActiveModel = m.into();
        m.output = Set(Some(output));
        m.percent = Set(percent);
        m.update(&tx).await?;
        tx.commit().await.map_err(Into::into)
    }
}

pub struct Service {
    pub killer_tx: RwLock<HashMap<HyUuid, oneshot::Sender<()>>>,
}

impl Service {
    pub async fn clean_running(&self, db: &DatabaseTransaction) -> Result<u64> {
        Ok(tasks::Entity::update_many()
            .col_expr(tasks::Column::Result, Expr::value(-1))
            .filter(tasks::Column::Result.is_null())
            .exec(db)
            .await?
            .rows_affected)
    }
}

#[async_trait]
impl skynet_api_task::Service for Service {
    async fn create(
        &self,
        name: &str,
        detail: Option<&str>,
        f: impl FnOnce(
                Box<dyn skynet_api_task::TaskItem>,
                oneshot::Receiver<()>,
            ) -> Pin<Box<dyn Future<Output = Result<i32>> + Send>>
            + Send
            + 'static,
    ) -> Result<HyUuid> {
        let tx = DB.get().unwrap().begin().await?;
        let m = tasks::ActiveModel {
            name: Set(name.to_owned()),
            detail: Set(detail.map(ToOwned::to_owned)),
            ..Default::default()
        }
        .insert(&tx)
        .await?;
        tx.commit().await?;
        let (tx, rx) = oneshot::channel();
        self.killer_tx.write().insert(m.id, tx);
        spawn(async move {
            let ret = f(Box::new(TaskItem::new(&m.id)), rx)
                .await
                .unwrap_or_else(|e| {
                    error!(id = %m.id, error = %e, "Task failed to execute");
                    -1
                });
            let tx = DB.get().unwrap().begin().await?;
            let mut m: tasks::ActiveModel = tasks::Entity::find_by_id(m.id)
                .one(&tx)
                .await?
                .ok_or(anyhow!("Task not found"))?
                .into();
            m.result = Set(Some(ret));
            m.update(&tx).await?;
            tx.commit().await?;
            Ok::<(), Error>(())
        });
        Ok(m.id)
    }

    async fn find(
        &self,
        db: &DatabaseTransaction,
        cond: Condition,
    ) -> Result<(Vec<tasks::Model>, u64)> {
        cond.select_page(tasks::Entity::find(), db).await
    }

    async fn find_by_id(
        &self,
        db: &DatabaseTransaction,
        id: &HyUuid,
    ) -> Result<Option<tasks::Model>> {
        tasks::Entity::find_by_id(id.to_owned())
            .one(db)
            .await
            .map_err(Into::into)
    }

    async fn delete_completed(&self, db: &DatabaseTransaction) -> Result<u64> {
        tasks::Entity::delete_many()
            .filter(tasks::Column::Result.is_not_null())
            .exec(db)
            .await
            .map(|x| x.rows_affected)
            .map_err(Into::into)
    }
}
