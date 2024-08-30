use actix::{Actor, ActorContext, Addr, AsyncContext, StreamHandler};
use actix_web_actors::ws::{self, start, ProtocolError, WebsocketContext};
use derivative::Derivative;
use monitor_api::{
    frontend_message::Data, message, prost::Message as _, FrontendMessage, ShellDisconnectMessage,
    ShellErrorMessage,
};
use skynet_api::{
    actix_cloud::{
        actix_web::{
            web::{Bytes, Payload},
            Error, HttpRequest, HttpResponse,
        },
        bail,
    },
    anyhow,
    tracing::{debug, Instrument},
    HyUuid, Result,
};
use skynet_macro::plugin_api;

use crate::{SERVICE, WEB_ADDRESS};

pub type WSAddr = Addr<WSHandler>;

#[derive(Derivative)]
pub struct WSHandler {
    pub id: Option<HyUuid>,
    pub token: HyUuid,
}

impl Actor for WSHandler {
    type Context = WebsocketContext<Self>;

    fn stopped(&mut self, _: &mut Self::Context) {
        self.cleanup();
    }
}

impl WSHandler {
    const fn new() -> Self {
        Self {
            id: None,
            token: HyUuid::nil(),
        }
    }

    fn cleanup(&mut self) {
        if self.id.is_some() {
            let _ = self.send_msg(message::Data::ShellDisconnect(ShellDisconnectMessage {
                token: Some(self.token.to_string()),
            }));
            WEB_ADDRESS.get().unwrap().write().remove(&self.token);
            debug!(token = %self.token, id = ?self.id, "Websocket cleanup");
        }
        self.id = None;
        self.token = HyUuid::nil();
    }

    fn send_msg(&self, data: message::Data) -> Result<()> {
        if let Some(id) = &self.id {
            if let Some(agent) = SERVICE.get().unwrap().agent.read().get(id) {
                if !agent.disable_shell {
                    let c = agent.message.as_ref().ok_or(anyhow!("Agent is offline"))?;
                    c.send(data)?;
                }
                Ok(())
            } else {
                bail!("Agent does not exist")
            }
        } else {
            bail!("Invalid message type")
        }
    }

    fn recv(&mut self, ctx: &mut <Self as Actor>::Context, text: Bytes) -> Result<()> {
        let msg = FrontendMessage::decode(text)?;
        if let Some(data) = msg.data {
            match data {
                Data::ShellConnect(data) => {
                    self.cleanup();
                    let token = HyUuid::parse(&data.token)?;
                    self.id = Some(HyUuid::parse(&msg.id.ok_or(anyhow!("Invalid message"))?)?);
                    self.token = token;
                    WEB_ADDRESS
                        .get()
                        .unwrap()
                        .write()
                        .insert(self.token, ctx.address());
                    debug!(token = %self.token, id = ?self.id, "Websocket shell connect");
                    self.send_msg(message::Data::ShellConnect(data))
                }
                Data::ShellInput(mut data) => {
                    data.token = Some(self.token.to_string());
                    self.send_msg(message::Data::ShellInput(data))
                }
                Data::ShellResize(mut data) => {
                    data.token = Some(self.token.to_string());
                    self.send_msg(message::Data::ShellResize(data))
                }
                Data::ShellDisconnect(_) => {
                    self.cleanup();
                    Ok(())
                }
                _ => bail!("Invalid message type"),
            }
        } else {
            bail!("Invalid data")
        }
    }
}

impl StreamHandler<Result<ws::Message, ProtocolError>> for WSHandler {
    fn handle(&mut self, msg: Result<ws::Message, ProtocolError>, ctx: &mut Self::Context) {
        match msg {
            Ok(ws::Message::Ping(msg)) => ctx.pong(&msg),
            Ok(ws::Message::Binary(text)) => {
                if let Err(e) = self.recv(ctx, text) {
                    ctx.binary(
                        FrontendMessage {
                            id: None,
                            data: Some(Data::ShellError(ShellErrorMessage {
                                token: None,
                                error: e.to_string(),
                            })),
                        }
                        .encode_to_vec(),
                    );
                    debug!(error = %e, "Error handle ws message");
                }
            }
            Ok(ws::Message::Close(reason)) => {
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
