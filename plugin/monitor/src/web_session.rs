use actix::{Handler, Message};
use monitor_service::{client, server};
use skynet::HyUuid;

use crate::{agent_ws, web_ws};

#[derive(Message)]
#[rtype(result = "()")]
pub struct ShellConnect {
    pub id: HyUuid,
    pub data: client::ShellConnect,
}

impl Handler<ShellConnect> for agent_ws::WSHandler {
    type Result = ();

    fn handle(&mut self, msg: ShellConnect, ctx: &mut Self::Context) -> Self::Result {
        ctx.text(client::Message {
            id: msg.id,
            data: client::DataType::ShellConnect(msg.data),
        });
    }
}

#[derive(Message)]
#[rtype(result = "()")]
pub struct ShellResize {
    pub data: client::ShellResize,
}

impl Handler<ShellResize> for agent_ws::WSHandler {
    type Result = ();

    fn handle(&mut self, msg: ShellResize, ctx: &mut Self::Context) -> Self::Result {
        ctx.text(client::Message::new(client::DataType::ShellResize(
            msg.data,
        )));
    }
}

#[derive(Message)]
#[rtype(result = "()")]
pub struct ShellInput {
    pub data: client::ShellInput,
}

impl Handler<ShellInput> for agent_ws::WSHandler {
    type Result = ();

    fn handle(&mut self, msg: ShellInput, ctx: &mut Self::Context) -> Self::Result {
        ctx.text(client::Message::new(client::DataType::ShellInput(msg.data)));
    }
}

#[derive(Message)]
#[rtype(result = "()")]
pub struct ShellDisconnect {
    pub data: client::ShellDisconnect,
}

impl Handler<ShellDisconnect> for agent_ws::WSHandler {
    type Result = ();

    fn handle(&mut self, msg: ShellDisconnect, ctx: &mut Self::Context) -> Self::Result {
        ctx.text(client::Message::new(client::DataType::ShellDisconnect(
            msg.data,
        )));
    }
}

#[derive(Message)]
#[rtype(result = "()")]
pub struct ShellConnectRsp {
    pub id: HyUuid,
    pub data: server::ShellConnect,
}

impl Handler<ShellConnectRsp> for web_ws::WSHandler {
    type Result = ();

    fn handle(&mut self, msg: ShellConnectRsp, ctx: &mut Self::Context) -> Self::Result {
        if let Some(conn) = &mut self.conn {
            conn.token = msg.data.token.clone();
        }
        ctx.text(server::Message {
            id: msg.id,
            data: server::DataType::ShellConnect(msg.data),
        });
    }
}

#[derive(Message)]
#[rtype(result = "()")]
pub struct ShellOutput {
    pub data: server::ShellOutput,
}

impl Handler<ShellOutput> for web_ws::WSHandler {
    type Result = ();

    fn handle(&mut self, mut msg: ShellOutput, ctx: &mut Self::Context) -> Self::Result {
        msg.data.token.clear(); // mask token
        ctx.text(server::Message::new(server::DataType::ShellOutput(
            msg.data,
        )));
    }
}
