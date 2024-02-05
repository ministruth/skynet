use actix::{Actor, ActorContext, Addr, AsyncContext, StreamHandler};
use actix_web_actors::ws::{start, Message, ProtocolError, WebsocketContext};
use derivative::Derivative;
use futures::executor::block_on;
use sea_orm_migration::sea_orm::TransactionTrait;
use skynet::{
    actix_web::{
        web::{Bytes, Data, Payload},
        Error, HttpRequest, HttpResponse,
    },
    anyhow::{self, bail, Result},
    log::{debug, info},
    request::Request,
    HyUuid, Skynet,
};

use crate::{
    msg::{client, server},
    TokenSrv, SERVICE,
};

pub type WSAddr = Addr<WSHandler>;

#[derive(Derivative)]
pub(crate) struct WSHandler {
    id: Option<HyUuid>,
    skynet: Data<Skynet>,
    request: Request,
}

impl Actor for WSHandler {
    type Context = WebsocketContext<Self>;

    fn stopped(&mut self, _: &mut Self::Context) {
        if let Some(x) = self.id {
            SERVICE.get().unwrap().agent.logout(&x);
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
        if TokenSrv::get(&self.skynet).unwrap() == msg.token {
            block_on(async {
                let tx = SERVICE.get().unwrap().db.begin().await?;
                self.id = Some(
                    if let Some(x) = SERVICE
                        .get()
                        .unwrap()
                        .agent
                        .login(
                            &tx,
                            ctx.address(),
                            msg.uid,
                            self.request.ip.ip().to_string(),
                        )
                        .await?
                    {
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

    fn info_msg(id: &HyUuid, msg: server::Info) -> Result<()> {
        block_on(async {
            let tx = SERVICE.get().unwrap().db.begin().await?;
            SERVICE
                .get()
                .unwrap()
                .agent
                .update(&tx, id, msg.os, msg.system, msg.machine, msg.hostname)
                .await?;
            tx.commit().await?;
            Ok::<(), anyhow::Error>(())
        })?;
        Ok(())
    }

    fn recv(&mut self, text: &Bytes, ctx: &mut <Self as Actor>::Context) -> Result<()> {
        let msg: server::Message = serde_json::from_slice(text)?;
        if let Some(id) = self.id {
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
                server::DataType::Info(x) => Self::info_msg(&id, x)?,
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
