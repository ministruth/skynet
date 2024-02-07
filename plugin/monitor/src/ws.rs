use std::sync::Arc;

use actix::{Actor, ActorContext, Addr, AsyncContext, StreamHandler};
use actix_web_actors::ws::{start, Message, ProtocolError, WebsocketContext};
use base64::{engine::general_purpose::STANDARD, Engine};
use derivative::Derivative;
use futures::executor::block_on;
use miniz_oxide::deflate::compress_to_vec;
use monitor_service::{client, server, PluginSrv, ID};
use sea_orm_migration::sea_orm::TransactionTrait;
use serde_json::json;
use skynet::{
    actix_web::{
        web::{Bytes, Data, Payload},
        Error, HttpRequest, HttpResponse,
    },
    anyhow::{self, bail, Result},
    log::{debug, info, warn},
    request::Request,
    HyUuid, Skynet,
};

use crate::{ADDRESS, DB, SERVICE};

pub type WSAddr = Addr<WSHandler>;

#[derive(Derivative)]
pub struct WSHandler {
    id: Option<HyUuid>,
    skynet: Data<Skynet>,
    request: Request,
}

impl Actor for WSHandler {
    type Context = WebsocketContext<Self>;

    fn stopped(&mut self, _: &mut Self::Context) {
        if let Some(x) = self.id {
            SERVICE.get().unwrap().logout(&x);
            ADDRESS.get().unwrap().write().remove(&x);
            info!("Agent `{}` logout", x);
        }
    }
}

impl WSHandler {
    fn new(skynet: Data<Skynet>, request: Request) -> Self {
        Self {
            id: None,
            skynet,
            request,
        }
    }

    fn login(
        &mut self,
        ctx: &mut <Self as Actor>::Context,
        msg_id: &HyUuid,
        msg: server::Login,
    ) -> Result<()> {
        if PluginSrv::get_setting(&self.skynet).is_some_and(|x| x == msg.token) {
            block_on(async {
                let tx = DB.get().unwrap().begin().await?;
                self.id = Some(
                    if let Some(x) = SERVICE
                        .get()
                        .unwrap()
                        .login(&tx, msg.uid, self.request.ip.ip().to_string())
                        .await?
                    {
                        ADDRESS.get().unwrap().write().insert(x, ctx.address());
                        x
                    } else {
                        ctx.text(client::Message::new_rsp(
                            msg_id,
                            client::DataType::Login(client::Login {
                                code: 2,
                                msg: "Already login".to_owned(),
                            }),
                        ));
                        bail!("Already login")
                    },
                );
                tx.commit().await?;
                Ok::<(), anyhow::Error>(())
            })?;

            ctx.text(
                client::Message::new_rsp(
                    msg_id,
                    client::DataType::Login(client::Login {
                        code: 0,
                        msg: "Login success".to_owned(),
                    }),
                )
                .json(),
            );
            info!("Agent `{}` login", self.id.unwrap());
        } else {
            ctx.text(client::Message::new_rsp(
                msg_id,
                client::DataType::Login(client::Login {
                    code: 1,
                    msg: "Invalid token".to_owned(),
                }),
            ));
        }
        Ok(())
    }

    fn info_msg(
        &self,
        ctx: &mut <Self as Actor>::Context,
        msg_id: &HyUuid,
        msg: server::Info,
    ) -> Result<()> {
        block_on(async {
            let tx = DB.get().unwrap().begin().await?;
            SERVICE
                .get()
                .unwrap()
                .update(
                    &tx,
                    &self.id.unwrap(),
                    msg.os.clone(),
                    msg.system,
                    Some(msg.arch.clone()),
                    msg.hostname,
                )
                .await?;
            tx.commit().await?;
            Ok::<(), anyhow::Error>(())
        })?;
        if let Some(x) = self
            .skynet
            .shared_api
            .get(&agent_service::ID)
            .and_then(|x| x.downcast_ref::<Arc<agent_service::PluginSrv>>())
        {
            if agent_service::PluginSrv::check_version(&msg.version) {
                if let Some(sys) = agent_service::System::parse(&msg.os.clone().unwrap_or_default())
                {
                    if let Some(arch) = agent_service::Arch::parse(&msg.arch) {
                        if let Some(data) = x.get_binary(sys.clone(), arch.clone()) {
                            SERVICE.get().unwrap().update_state(&self.id.unwrap());
                            let crc = crc32fast::hash(&data);
                            let data = STANDARD.encode(compress_to_vec(&data, 6));
                            ctx.text(client::Message::new_rsp(
                                msg_id,
                                client::DataType::Update(client::Update {
                                    data: data,
                                    crc32: crc,
                                }),
                            ));
                        } else {
                            warn!(
                                "Agent not update, file not found\n{}",
                                json!({
                                    "plugin": ID,
                                    "aid": self.id.unwrap(),
                                    "file": x.get_binary_name(sys, arch),
                                })
                            )
                        }
                    } else {
                        warn!(
                            "Agent not update, arch invalid\n{}",
                            json!({
                                "plugin": ID,
                                "aid": self.id.unwrap(),
                                "arch": msg.arch,
                            })
                        )
                    }
                } else {
                    warn!(
                        "Agent not update, system invalid\n{}",
                        json!({
                            "plugin": ID,
                            "aid": self.id.unwrap(),
                            "system": msg.os.unwrap_or_default(),
                        })
                    )
                }
            }
        }
        Ok(())
    }

    fn recv(&mut self, text: &Bytes, ctx: &mut <Self as Actor>::Context) -> Result<()> {
        let msg: server::Message = serde_json::from_slice(text)?;
        if self.id.is_some() {
            match msg.data {
                server::DataType::Login(_) => {
                    ctx.text(client::Message::new_rsp(
                        &msg.id,
                        client::DataType::Login(client::Login {
                            code: 2,
                            msg: "Already login".to_owned(),
                        }),
                    ));
                }
                server::DataType::Info(x) => self.info_msg(ctx, &msg.id, x)?,
            }
        } else {
            match msg.data {
                server::DataType::Login(x) => self.login(ctx, &msg.id, x)?,
                _ => bail!("Invalid message, need login"),
            }
        }
        Ok(())
    }
}

impl StreamHandler<Result<Message, ProtocolError>> for WSHandler {
    fn handle(&mut self, msg: Result<Message, ProtocolError>, ctx: &mut Self::Context) {
        match msg {
            Ok(Message::Ping(msg)) => ctx.pong(&msg),
            Ok(Message::Text(text)) => {
                if let Err(e) = self.recv(text.as_bytes(), ctx) {
                    debug!("{e}");
                }
            }
            Ok(Message::Close(reason)) => {
                ctx.close(reason);
                ctx.stop();
            }
            _ => ctx.stop(),
        }
    }
}

pub async fn get(
    req: HttpRequest,
    r: Request,
    skynet: Data<Skynet>,
    payload: Payload,
) -> Result<HttpResponse, Error> {
    start(WSHandler::new(skynet, r), &req, payload)
}
