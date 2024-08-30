use std::net::SocketAddr;
use std::sync::Arc;
use std::{cmp::max, collections::HashMap};

use itertools::Itertools;
use once_cell::sync::Lazy;
use serde_json::Value;
use skynet_api::actix_cloud::anyhow;
use skynet_api::actix_cloud::chrono::Utc;
use skynet_api::actix_cloud::tokio::sync::mpsc::{unbounded_channel, UnboundedReceiver};
use skynet_api::hyuuid::uuids2strings;
use skynet_api::request::Condition;
use skynet_api::sea_orm::Unchanged;
use skynet_api::{
    async_trait,
    parking_lot::RwLock,
    sea_orm::{ActiveModelTrait, ColumnTrait, DatabaseTransaction, EntityTrait, QueryFilter, Set},
    HyUuid, Result, Skynet,
};
use skynet_api_monitor::entity::passive_agents;
use skynet_api_monitor::message::Data;
use skynet_api_monitor::{ecies::SecretKey, entity::agents, Agent, AgentStatus, ID};
use skynet_api_monitor::{InfoMessage, StatusRspMessage};

static SETTING_ADDRESS: Lazy<String> = Lazy::new(|| format!("plugin_{ID}_address"));
static SETTING_CERTIFICATE: Lazy<String> = Lazy::new(|| format!("plugin_{ID}_certificate"));
static SETTING_SHELL: Lazy<String> = Lazy::new(|| format!("plugin_{ID}_shell"));

pub struct Service {
    pub server: Arc<Box<dyn skynet_api_monitor::Server>>,
    pub view_id: HyUuid,
    pub manage_id: HyUuid,
    pub agent: Arc<RwLock<HashMap<HyUuid, Agent>>>,
}

impl Service {
    pub fn agent_exist(&self, id: &HyUuid) -> bool {
        self.agent.read().get(id).is_some()
    }

    pub async fn init(&self, db: &DatabaseTransaction) -> Result<()> {
        let agents: Vec<Agent> = agents::Entity::find()
            .all(db)
            .await?
            .into_iter()
            .map(From::from)
            .collect();
        let mut wlock = self.agent.write();
        for x in agents {
            wlock.insert(x.id, x);
        }

        Ok(())
    }

    pub async fn create_passive(
        &self,
        db: &DatabaseTransaction,
        name: &str,
        address: &str,
        retry_time: i32,
    ) -> Result<passive_agents::Model> {
        let ret = passive_agents::ActiveModel {
            name: Set(name.to_owned()),
            address: Set(address.to_owned()),
            retry_time: Set(retry_time),
            ..Default::default()
        }
        .insert(db)
        .await?;
        self.server.connect(&ret.id);
        Ok(ret)
    }

    pub async fn delete_passive(&self, db: &DatabaseTransaction, paid: &[HyUuid]) -> Result<u64> {
        passive_agents::Entity::delete_many()
            .filter(passive_agents::Column::Id.is_in(uuids2strings(paid)))
            .exec(db)
            .await
            .map(|x| x.rows_affected)
            .map_err(anyhow::Error::from)
    }

    pub async fn find_passive(
        &self,
        db: &DatabaseTransaction,
        cond: Condition,
    ) -> Result<(Vec<passive_agents::Model>, u64)> {
        cond.select_page(passive_agents::Entity::find(), db).await
    }

    pub async fn find_passive_by_id(
        &self,
        db: &DatabaseTransaction,
        id: &HyUuid,
    ) -> Result<Option<passive_agents::Model>> {
        passive_agents::Entity::find()
            .filter(passive_agents::Column::Id.eq(*id))
            .one(db)
            .await
            .map_err(anyhow::Error::from)
    }

    pub async fn find_passive_by_name(
        &self,
        db: &DatabaseTransaction,
        name: &str,
    ) -> Result<Option<passive_agents::Model>> {
        passive_agents::Entity::find()
            .filter(passive_agents::Column::Name.eq(name))
            .one(db)
            .await
            .map_err(anyhow::Error::from)
    }

    pub async fn find_passive_by_address(
        &self,
        db: &DatabaseTransaction,
        address: &str,
    ) -> Result<Option<passive_agents::Model>> {
        passive_agents::Entity::find()
            .filter(passive_agents::Column::Address.eq(address))
            .one(db)
            .await
            .map_err(anyhow::Error::from)
    }

    /// Find agent by `name`.
    /// Returns `None` when not found.
    ///
    /// # Errors
    ///
    /// Will raise `Err` for db errors.
    pub async fn find_by_name(
        &self,
        db: &DatabaseTransaction,
        name: &str,
    ) -> Result<Option<agents::Model>> {
        agents::Entity::find()
            .filter(agents::Column::Name.eq(name))
            .one(db)
            .await
            .map_err(anyhow::Error::from)
    }

    /// Find agent by `uid`.
    /// Returns `None` when not found.
    ///
    /// # Errors
    ///
    /// Will raise `Err` for db errors.
    pub async fn find_by_uid(
        &self,
        db: &DatabaseTransaction,
        uid: &str,
    ) -> Result<Option<agents::Model>> {
        agents::Entity::find()
            .filter(agents::Column::Uid.eq(uid))
            .one(db)
            .await
            .map_err(anyhow::Error::from)
    }

    /// Rename agent `id` name to `name`.
    ///
    /// # Errors
    ///
    /// Will raise `Err` for db errors.
    pub async fn rename(&self, db: &DatabaseTransaction, id: &HyUuid, name: &str) -> Result<()> {
        agents::ActiveModel {
            id: Unchanged(*id),
            name: Set(name.to_owned()),
            ..Default::default()
        }
        .update(db)
        .await?;
        if let Some(x) = self.agent.write().get_mut(id) {
            x.name = name.to_owned();
        }
        Ok(())
    }

    /// Delete agent `id`.
    /// Returns affected rows.
    ///
    /// # Errors
    ///
    /// Will raise `Err` for db errors.
    pub async fn delete(&self, db: &DatabaseTransaction, id: &HyUuid) -> Result<u64> {
        let num = agents::Entity::delete_by_id(*id)
            .exec(db)
            .await?
            .rows_affected;
        self.agent.write().remove(id);
        Ok(num)
    }

    /// Login agent `uid` with `ip`. Returns `None` when already login, otherwise agent id.
    ///
    /// # Errors
    ///
    /// Will raise `Err` for db errors.
    pub async fn login(
        &self,
        db: &DatabaseTransaction,
        uid: String,
        addr: SocketAddr,
    ) -> Result<Option<HyUuid>> {
        let ip = addr.ip().to_string();
        let agent = self.find_by_uid(db, &uid).await?;
        let now = Utc::now().timestamp_millis();
        let agent = if let Some(agent) = agent {
            agent
        } else {
            agents::ActiveModel {
                uid: Set(uid.clone()),
                name: Set(uid.chars().take(8).collect()),
                ip: Set(ip.clone()),
                last_login: Set(now),
                ..Default::default()
            }
            .insert(db)
            .await?
        };
        let status = self.agent.read().get(&agent.id).map(|x| x.status);
        if let Some(status) = status {
            if status.is_offline() {
                let mut agent: agents::ActiveModel = agent.into();
                agent.ip = Set(ip.clone());
                agent.last_login = Set(now);
                let agent = agent.update(db).await?;

                Ok(Some(
                    self.agent
                        .write()
                        .get_mut(&agent.id)
                        .map(|x| {
                            x.ip = ip;
                            x.last_login = now;
                            x.status = AgentStatus::Online;
                            x.address = Some(addr);
                            agent.id
                        })
                        .unwrap(),
                ))
            } else {
                Ok(None)
            }
        } else {
            let mut agent: Agent = agent.into();
            agent.status = AgentStatus::Online;
            agent.address = Some(addr);
            let id = agent.id;
            self.agent.write().insert(id, agent);
            Ok(Some(id))
        }
    }

    /// Logout agent `id`. Will be invoked automatically when connection losts.
    pub fn logout(&self, id: &HyUuid) {
        if let Some(item) = self.agent.write().get_mut(id) {
            item.status = AgentStatus::Offline;
            item.endpoint.clear();
            item.address = None;
            item.disable_shell = false;
            item.report_rate = 0;
            item.last_rsp = None;
            item.cpu = None;
            item.memory = None;
            item.total_memory = None;
            item.disk = None;
            item.total_disk = None;
            item.latency = None;
            item.net_up = None;
            item.net_down = None;
            item.band_up = None;
            item.band_down = None;
            item.message = None;
        }
    }

    /// Update agent `id` status.
    pub fn update_status(&self, id: &HyUuid, data: StatusRspMessage) {
        let now = Utc::now().timestamp_millis();
        if let Some(item) = self.agent.write().get_mut(id) {
            if let Some(rsp) = item.last_rsp {
                if let Some(x) = item.band_up {
                    item.net_up = Some((data.band_up - x) * 1000 / max(now - rsp, 1) as u64);
                }
                if let Some(x) = item.band_down {
                    item.net_down = Some((data.band_down - x) * 1000 / max(now - rsp, 1) as u64);
                }
            }

            item.last_rsp = Some(now);
            item.cpu = Some(data.cpu);
            item.memory = Some(data.memory);
            item.total_memory = Some(data.total_memory);
            item.disk = Some(data.disk);
            item.total_disk = Some(data.total_disk);
            item.latency = Some((now - data.time) / 2); // round trip
            item.band_up = Some(data.band_up);
            item.band_down = Some(data.band_down);
        }
    }

    /// Update agent `id` with infos.
    ///
    /// # Errors
    ///
    /// Will raise `Err` for db errors.
    pub async fn update(
        &self,
        db: &DatabaseTransaction,
        id: &HyUuid,
        data: InfoMessage,
    ) -> Result<()> {
        agents::ActiveModel {
            id: Unchanged(*id),
            os: Set(data.os.clone()),
            system: Set(data.system.clone()),
            arch: Set(data.arch.clone()),
            hostname: Set(data.hostname.clone()),
            ..Default::default()
        }
        .update(db)
        .await?;
        if let Some(item) = self.agent.write().get_mut(id) {
            item.os = data.os;
            item.system = data.system;
            item.arch = data.arch;
            item.hostname = data.hostname;
            if let Some(ip) = data.ip {
                item.ip = ip;
            }
            item.endpoint = data.endpoint;
            item.disable_shell = data.disable_shell;
            item.report_rate = data.report_rate;
        }
        Ok(())
    }

    /// Bind message channel.
    pub fn bind_message(&self, id: &HyUuid) -> UnboundedReceiver<Data> {
        let (tx, rx) = unbounded_channel();
        if let Some(item) = self.agent.write().get_mut(id) {
            item.message = Some(tx);
        }
        rx
    }
}

#[async_trait]
impl skynet_api_monitor::Service for Service {
    fn get_server(&self) -> Arc<Box<dyn skynet_api_monitor::Server>> {
        self.server.clone()
    }

    fn get_view_id(&self) -> HyUuid {
        self.view_id
    }

    fn get_manage_id(&self) -> HyUuid {
        self.manage_id
    }

    fn get_agents(&self) -> &RwLock<HashMap<HyUuid, Agent>> {
        &self.agent
    }

    fn get_setting_address(&self, skynet: &Skynet) -> Option<String> {
        skynet.setting.get(&SETTING_ADDRESS)
    }

    fn get_setting_certificate(&self, skynet: &Skynet) -> Option<SecretKey> {
        skynet
            .setting
            .get_base64(&SETTING_CERTIFICATE)
            .and_then(|d| d.try_into().ok().and_then(|d| SecretKey::parse(&d).ok()))
    }

    fn get_setting_shell(&self, skynet: &Skynet) -> Option<Vec<String>> {
        if let Some(x) = skynet.setting.get(&SETTING_SHELL) {
            if let Ok(x) = serde_json::from_str::<Value>(&x) {
                return x.as_array().map(|x| {
                    x.iter()
                        .map(|x| x.as_str().unwrap_or("").to_owned())
                        .unique()
                        .filter(|x| !x.is_empty())
                        .collect()
                });
            }
        }
        None
    }

    async fn set_setting_address(
        &self,
        db: &DatabaseTransaction,
        skynet: &Skynet,
        address: &str,
    ) -> Result<()> {
        skynet.setting.set(db, &SETTING_ADDRESS, address).await
    }

    async fn set_setting_certificate(
        &self,
        db: &DatabaseTransaction,
        skynet: &Skynet,
        cert: &SecretKey,
    ) -> Result<()> {
        skynet
            .setting
            .set_base64(db, &SETTING_CERTIFICATE, &cert.serialize())
            .await
    }

    async fn set_setting_shell(
        &self,
        db: &DatabaseTransaction,
        skynet: &Skynet,
        shell_prog: &[String],
    ) -> Result<()> {
        skynet
            .setting
            .set(db, &SETTING_SHELL, &serde_json::to_string(&shell_prog)?)
            .await
    }
}
