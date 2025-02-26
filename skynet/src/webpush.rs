use std::{io::Cursor, sync::Arc, thread};

use actix_cloud::tokio::{
    runtime,
    sync::mpsc::{UnboundedSender, unbounded_channel},
};
use dashmap::DashMap;
use enum_map::EnumMap;
use parking_lot::RwLock;
use reqwest::{
    StatusCode,
    header::{CONTENT_ENCODING, CONTENT_LENGTH, CONTENT_TYPE},
};
use serde_json::json;
use skynet_api::{
    HyUuid, Result,
    config::CONFIG_WEBPUSH_KEY,
    entity::{webpush_clients, webpush_subscriptions},
    permission::{
        IDTypes::{self, *},
        PERM_READ, PermChecker,
    },
    request::Condition,
    sea_orm::{ColumnTrait, DatabaseConnection},
    service::Message,
    tracing::debug,
    uuid,
    viewer::{
        settings::SettingViewer, webpush_clients::WebpushClientViewer,
        webpush_subscriptions::WebpushSubscriptionViewer,
    },
};
use web_push::{ContentEncoding, SubscriptionInfo, VapidSignatureBuilder, WebPushMessageBuilder};

use crate::service;

#[derive(Debug)]
pub struct Topic {
    pub id: HyUuid,
    pub name: String,
    pub perm: PermChecker,
}

impl Topic {
    pub async fn get_users(&self, db: &DatabaseConnection) -> Result<Vec<HyUuid>> {
        let s = WebpushSubscriptionViewer::find(
            db,
            Condition::new(Condition::all().add(webpush_subscriptions::Column::Topic.eq(self.id))),
        )
        .await?
        .0;
        let mut ret = Vec::new();
        for i in s {
            let perm = service::get_user_perm(db, &i.uid).await?;
            if !self.perm.check(&perm) {
                WebpushSubscriptionViewer::delete(db, &[i.id]).await?;
            } else {
                ret.push(i.uid);
            }
        }
        Ok(ret)
    }
}

#[derive(Debug, Clone)]
pub struct WebpushManager {
    pub topic: Arc<DashMap<HyUuid, Arc<Topic>>>,
    tx: UnboundedSender<(HyUuid, Message)>,
    pub key: Arc<RwLock<Vec<u8>>>,
}

impl WebpushManager {
    pub async fn new(db: DatabaseConnection, id: &EnumMap<IDTypes, HyUuid>) -> Result<Self> {
        let key = Arc::new(RwLock::new(
            SettingViewer::get_base64(&db, CONFIG_WEBPUSH_KEY)
                .await?
                .unwrap_or_default(),
        ));
        let topic: Arc<DashMap<HyUuid, Arc<Topic>>> = Default::default();
        let (tx, mut rx) = unbounded_channel::<(HyUuid, Message)>();
        let t_key = key.clone();
        let t_topic = topic.clone();
        let ret = Self { topic, tx, key };

        ret.add_topic(
            HyUuid(uuid!("9338710c-5d89-434e-8a1f-b4eaa8446514")),
            String::from("skynet.notification"),
            PermChecker::new_entry(id[PermManageNotificationID], PERM_READ),
        )
        .await;

        thread::spawn(move || {
            runtime::Builder::new_current_thread()
                .enable_all()
                .build()
                .unwrap()
                .block_on(async move {
                    debug!("Webpush loop enter");
                    loop {
                        let m = match rx.recv().await {
                            Some(x) => x,
                            None => {
                                debug!("Webpush loop exit");
                                break;
                            }
                        };
                        let t = match t_topic.get(&m.0) {
                            Some(x) => x.clone(),
                            None => {
                                debug!("Topic id does not exist");
                                continue;
                            }
                        };
                        match t.get_users(&db).await {
                            Ok(users) => {
                                let key = t_key.read().clone();
                                for i in users {
                                    let client = match WebpushClientViewer::find(
                                        &db,
                                        Condition::new(
                                            Condition::all()
                                                .add(webpush_clients::Column::Uid.eq(i)),
                                        ),
                                    )
                                    .await
                                    {
                                        Ok(x) => x.0,
                                        Err(e) => {
                                            debug!("{e}");
                                            continue;
                                        }
                                    };
                                    for c in client {
                                        let ret = Self::push(
                                            &key,
                                            &SubscriptionInfo::new(&c.endpoint, &c.p256dh, &c.auth),
                                            &m.1,
                                        )
                                        .await;
                                        match ret {
                                            Ok(ret) => {
                                                if !ret {
                                                    let _ =
                                                        WebpushClientViewer::delete(&db, &[c.id])
                                                            .await;
                                                }
                                            }
                                            Err(e) => debug!("{e}"),
                                        }
                                    }
                                }
                            }
                            Err(e) => {
                                debug!("{e}");
                                continue;
                            }
                        }
                    }
                })
        });
        Ok(ret)
    }

    pub async fn add_topic(
        &self,
        topic: HyUuid,
        name: String,
        perm: PermChecker,
    ) -> Option<Arc<Topic>> {
        debug!(%topic, name, "New topic add");
        self.topic.insert(
            topic,
            Arc::new(Topic {
                id: topic,
                name,
                perm,
            }),
        )
    }

    pub fn send(&self, topic: HyUuid, message: Message) {
        let _ = self.tx.send((topic, message));
    }

    async fn push(key: &[u8], info: &SubscriptionInfo, message: &Message) -> Result<bool> {
        let title = if message.title.is_empty() {
            String::from("Skynet")
        } else {
            message.title.clone()
        };
        let content = serde_json::to_string(&json!({
            "title": title,
            "body": message.body,
            "url": message.url,
        }))?;
        let mut builder = WebPushMessageBuilder::new(info);
        builder.set_payload(ContentEncoding::Aes128Gcm, content.as_bytes());
        builder
            .set_vapid_signature(VapidSignatureBuilder::from_pem(Cursor::new(key), info)?.build()?);
        let message = builder.build()?;

        let client = reqwest::Client::new();
        let mut req = client
            .post(message.endpoint.to_string())
            .header("TTL", format!("{}", message.ttl).as_bytes());
        if let Some(urgency) = message.urgency {
            req = req.header("Urgency", urgency.to_string());
        }
        if let Some(topic) = message.topic {
            req = req.header("Topic", topic);
        }
        if let Some(payload) = message.payload {
            req = req
                .header(CONTENT_ENCODING, payload.content_encoding.to_str())
                .header(
                    CONTENT_LENGTH,
                    format!("{}", payload.content.len() as u64).as_bytes(),
                )
                .header(CONTENT_TYPE, "application/octet-stream");
            for (k, v) in payload.crypto_headers.into_iter() {
                let v: &str = v.as_ref();
                req = req.header(k, v);
            }
            req = req.body(payload.content);
        } else {
            req = req.body("")
        }
        let rsp = req.send().await?;
        if rsp.status().is_client_error() && rsp.status() != StatusCode::TOO_MANY_REQUESTS {
            Ok(false)
        } else {
            Ok(true)
        }
    }
}
