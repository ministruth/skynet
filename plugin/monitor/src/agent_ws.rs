use std::{sync::Arc, time::Duration};

use actix::{Actor, ActorContext, Addr, AsyncContext, StreamHandler};
use actix_web_actors::ws::{start, Message, ProtocolError, WebsocketContext};
use base64::{engine::general_purpose::STANDARD, Engine};
use derivative::Derivative;
use miniz_oxide::deflate::compress_to_vec;
use monitor_service::{client, server, PluginSrv, ID};
use sea_orm_migration::sea_orm::TransactionTrait;
use skynet::{
    actix_web::{
        web::{Bytes, Data, Payload},
        Error, HttpRequest, HttpResponse,
    },
    anyhow::{self, bail, Result},
    request::Request,
    tracing::{debug, info, warn},
    HyUuid, Skynet, Utc,
};
use skynet_macro::{plugin_api, tracing_api};

use crate::{
    web_session::{ShellConnectRsp, ShellOutput},
    AGENT_ADDRESS, DB, RUNTIME, SERVICE, WEB_ADDRESS,
};

pub type WSAddr = Addr<WSHandler>;

#[derive(Derivative)]
pub struct WSHandler {
    id: Option<HyUuid>,
    skynet: Data<Skynet>,
    request: Request,
}

impl Actor for WSHandler {
    type Context = WebsocketContext<Self>;

    #[tracing_api(self.request.request_id, self.request.ip)]
    fn stopped(&mut self, _: &mut Self::Context) {
        if let Some(x) = self.id {
            SERVICE.get().unwrap().logout(&x);
            AGENT_ADDRESS.get().unwrap().write().remove(&x);
            info!(aid = %x, "Agent logout");
        }
    }
}

impl WSHandler {
    const fn new(skynet: Data<Skynet>, request: Request) -> Self {
        Self {
            id: None,
            skynet,
            request,
        }
    }

    #[allow(clippy::unused_self)]
    fn send_status(&mut self, ctx: &mut <Self as Actor>::Context) {
        ctx.text(client::Message::new(client::DataType::Status(
            client::Status {
                time: Utc::now().timestamp_millis(),
            },
        )));
    }

    fn login(
        &mut self,
        ctx: &mut <Self as Actor>::Context,
        msg_id: &HyUuid,
        msg: server::Login,
    ) -> Result<()> {
        if PluginSrv::get_token(&self.skynet).is_some_and(|x| x == msg.token) {
            RUNTIME.get().unwrap().block_on(async {
                let tx = DB.get().unwrap().begin().await?;
                self.id = Some(
                    if let Some(x) = SERVICE
                        .get()
                        .unwrap()
                        .login(&tx, msg.uid, self.request.ip.ip().to_string())
                        .await?
                    {
                        AGENT_ADDRESS
                            .get()
                            .unwrap()
                            .write()
                            .insert(x, ctx.address());
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
                ctx.run_interval(Duration::from_secs(1), Self::send_status);
                Ok(())
            })?;

            ctx.text(client::Message::new_rsp(
                msg_id,
                client::DataType::Login(client::Login {
                    code: 0,
                    msg: "Login success".to_owned(),
                }),
            ));
            info!(aid = %self.id.unwrap(), "Agent login");
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
        RUNTIME.get().unwrap().block_on(async {
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
                let sys = agent_service::System::parse(&msg.os.clone().unwrap_or_default());
                let arch = agent_service::Arch::parse(&msg.arch);
                if sys.is_none() || arch.is_none() {
                    warn!(
                        plugin = %ID,
                        aid = %self.id.unwrap(),
                        arch = msg.arch,
                        system = msg.os,
                        "Agent not update, platform invalid",
                    );
                }

                if let Some(data) = x.get_binary(sys.unwrap(), arch.unwrap()) {
                    SERVICE.get().unwrap().update_state(&self.id.unwrap());
                    let crc = crc32fast::hash(&data);
                    let data = STANDARD.encode(compress_to_vec(&data, 6));
                    ctx.text(client::Message::new_rsp(
                        msg_id,
                        client::DataType::Update(client::Update { crc32: crc, data }),
                    ));
                } else {
                    warn!(
                        plugin = %ID,
                        aid = %self.id.unwrap(),
                        file = %x.get_binary_name(sys.unwrap(), arch.unwrap()).to_string_lossy(),
                        "Agent not update, file not found",
                    );
                }
            }
        }
        Ok(())
    }

    fn status_msg(&self, msg: &server::Status) {
        SERVICE.get().unwrap().update_status(
            &self.id.unwrap(),
            msg.time,
            msg.cpu,
            msg.memory,
            msg.total_memory,
            msg.disk,
            msg.total_disk,
            msg.band_up,
            msg.band_down,
        );
    }

    fn shell_connect(
        &mut self,
        ctx: &mut <Self as Actor>::Context,
        id: &HyUuid,
        msg: server::ShellConnect,
    ) {
        let id_string = &id.to_string();
        let mut lock = WEB_ADDRESS.get().unwrap().write();
        if let Some(addr) = lock.remove(id_string) {
            let token = msg.token.clone();
            addr.do_send(ShellConnectRsp { id: *id, data: msg });
            if !token.is_empty() {
                lock.insert(token.clone(), addr);
                info!(success = true, aid = %self.id.unwrap(), token = &token[..8], "Shell connected");
            }
            drop(lock);
        } else {
            drop(lock);
            // clean up shell
            ctx.text(client::Message::new(client::DataType::ShellDisconnect(
                client::ShellDisconnect { token: msg.token },
            )));
        }
    }

    #[allow(clippy::unused_self)]
    fn shell_output(
        &mut self,
        _ctx: &mut <Self as Actor>::Context,
        _id: &HyUuid,
        msg: server::ShellOutput,
    ) {
        if let Some(addr) = WEB_ADDRESS.get().unwrap().read().get(&msg.token) {
            addr.do_send(ShellOutput { data: msg });
        }
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
                server::DataType::Status(x) => self.status_msg(&x),
                server::DataType::ShellConnect(x) => self.shell_connect(ctx, &msg.id, x),
                server::DataType::ShellOutput(x) => self.shell_output(ctx, &msg.id, x),
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
    #[tracing_api(self.request.request_id, self.request.ip)]
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

#[plugin_api]
pub async fn service(
    req: HttpRequest,
    r: Request,
    skynet: Data<Skynet>,
    payload: Payload,
) -> Result<HttpResponse, Error> {
    start(WSHandler::new(skynet, r), &req, payload)
}
