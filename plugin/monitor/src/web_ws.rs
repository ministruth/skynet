use actix::{Actor, ActorContext, Addr, AsyncContext, StreamHandler};
use actix_web_actors::ws::{start, Message, ProtocolError, WebsocketContext};
use derivative::Derivative;
use monitor_service::client;
use skynet::{
    actix_web::{
        web::{Bytes, Payload},
        Error, HttpRequest, HttpResponse,
    },
    anyhow::{bail, Result},
    tracing::debug,
    HyUuid,
};
use skynet_macro::plugin_api;

use crate::{
    web_session::{ShellConnect, ShellDisconnect, ShellInput, ShellResize},
    AGENT_ADDRESS, WEB_ADDRESS,
};

pub type WSAddr = Addr<WSHandler>;

pub struct WebWSConnection {
    pub id: HyUuid,
    pub aid: HyUuid,
    pub token: String,
}

#[derive(Derivative)]
pub struct WSHandler {
    pub conn: Option<WebWSConnection>,
}

impl Actor for WSHandler {
    type Context = WebsocketContext<Self>;

    fn stopped(&mut self, _: &mut Self::Context) {
        self.cleanup();
    }
}

impl WSHandler {
    const fn new() -> Self {
        Self { conn: None }
    }

    fn cleanup(&mut self) {
        if let Some(conn) = &self.conn {
            WEB_ADDRESS
                .get()
                .unwrap()
                .write()
                .remove(&conn.id.to_string());
            WEB_ADDRESS.get().unwrap().write().remove(&conn.token);
            if let Some(addr) = AGENT_ADDRESS.get().unwrap().read().get(&conn.aid) {
                addr.do_send(ShellDisconnect {
                    data: client::ShellDisconnect {
                        token: conn.token.clone(),
                    },
                });
            }
        }
        self.conn = None;
    }

    fn recv(&mut self, text: &Bytes, ctx: &mut <Self as Actor>::Context) -> Result<()> {
        let msg: client::Message = serde_json::from_slice(text)?;
        match msg.data {
            client::DataType::ShellConnect(data) => {
                self.cleanup();
                if let Some(addr) = AGENT_ADDRESS.get().unwrap().read().get(&data.id) {
                    WEB_ADDRESS
                        .get()
                        .unwrap()
                        .write()
                        .insert(msg.id.to_string(), ctx.address());
                    self.conn = Some(WebWSConnection {
                        id: msg.id,
                        aid: data.id,
                        token: String::new(),
                    });
                    addr.do_send(ShellConnect { id: msg.id, data });
                } else {
                    ctx.stop();
                    bail!("Agent not found or offline")
                }
            }
            client::DataType::ShellResize(data) => {
                if let Some(conn) = &self.conn {
                    if !conn.token.is_empty() && conn.token == data.token {
                        if let Some(addr) = AGENT_ADDRESS.get().unwrap().read().get(&conn.aid) {
                            addr.do_send(ShellResize { data });
                        } else {
                            self.cleanup();
                            ctx.stop();
                        }
                    }
                }
            }
            client::DataType::ShellInput(data) => {
                if let Some(conn) = &self.conn {
                    if !conn.token.is_empty() && conn.token == data.token {
                        if let Some(addr) = AGENT_ADDRESS.get().unwrap().read().get(&conn.aid) {
                            addr.do_send(ShellInput { data });
                        } else {
                            self.cleanup();
                            ctx.stop();
                        }
                    }
                }
            }
            client::DataType::ShellDisconnect(_) => {
                self.cleanup();
            }
            _ => bail!("Invalid message {:?}", text),
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

#[plugin_api]
pub async fn service(req: HttpRequest, payload: Payload) -> Result<HttpResponse, Error> {
    start(WSHandler::new(), &req, payload)
}
