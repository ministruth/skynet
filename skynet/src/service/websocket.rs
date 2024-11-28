use actix_cloud::tokio::sync::mpsc::{channel, Receiver, Sender};
use actix_ws::Session;
use dashmap::DashMap;
use skynet_api::{
    ffi_rpc::{
        self, abi_stable, async_trait, bincode,
        ffi_rpc_macro::{plugin_impl_instance, plugin_impl_trait},
        registry::Registry,
    },
    plugin::{PluginError, WSMessage},
    service::SResult,
    HyUuid,
};

#[plugin_impl_instance(|| WebsocketImpl{ sessions:Default::default() })]
pub struct WebsocketImpl {
    sessions: DashMap<HyUuid, (Session, Sender<bool>)>,
}

impl WebsocketImpl {
    pub fn add(&self, id: HyUuid, session: Session) -> Receiver<bool> {
        let (tx, rx) = channel(1);
        self.sessions.insert(id, (session, tx));
        rx
    }

    pub fn remove(&self, id: &HyUuid) {
        self.sessions.remove(id);
    }
}

#[plugin_impl_trait]
impl skynet_api::service::skynet::Websocket for WebsocketImpl {
    async fn send(&self, _: &Registry, id: HyUuid, msg: WSMessage) -> SResult<()> {
        match msg {
            WSMessage::Text(s) => {
                let x = self.sessions.get(&id).map(|x| x.0.clone());
                match x {
                    Some(mut x) => x.text(s).await.or(Err(PluginError::SessionClosed.into())),
                    None => {
                        self.sessions.remove(&id);
                        Err(PluginError::SessionNotFound.into())
                    }
                }
            }
            WSMessage::Binary(s) => {
                let x = self.sessions.get(&id).map(|x| x.0.clone());
                match x {
                    Some(mut x) => x.binary(s).await.or(Err(PluginError::SessionClosed.into())),
                    None => {
                        self.sessions.remove(&id);
                        Err(PluginError::SessionNotFound.into())
                    }
                }
            }
            WSMessage::Close => {
                let (_, x) = self
                    .sessions
                    .remove(&id)
                    .ok_or(PluginError::SessionNotFound)?;
                x.0.close(None)
                    .await
                    .or(Err(PluginError::SessionClosed.into()))
            }
            _ => Ok(()),
        }
    }

    async fn close(&self, _: &Registry, id: HyUuid) {
        let x = self.sessions.remove(&id);
        if let Some((_, x)) = x {
            let _ = x.0.close(None).await;
            let _ = x.1.send(true).await;
        };
    }
}
