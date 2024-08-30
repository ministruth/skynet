use actix::{Handler, Message};
use skynet_api_monitor::{
    frontend_message::Data, prost::Message as _, FrontendMessage, ShellErrorMessage,
    ShellOutputMessage,
};

use crate::ws;

#[derive(Message)]
#[rtype(result = "()")]
pub struct ShellError {
    pub data: ShellErrorMessage,
}

impl Handler<ShellError> for ws::WSHandler {
    type Result = ();

    fn handle(&mut self, msg: ShellError, ctx: &mut Self::Context) -> Self::Result {
        ctx.binary(
            FrontendMessage {
                id: None,
                data: Some(Data::ShellError(msg.data)),
            }
            .encode_to_vec(),
        );
    }
}

#[derive(Message)]
#[rtype(result = "()")]
pub struct ShellOutput {
    pub data: ShellOutputMessage,
}

impl Handler<ShellOutput> for ws::WSHandler {
    type Result = ();

    fn handle(&mut self, msg: ShellOutput, ctx: &mut Self::Context) -> Self::Result {
        ctx.binary(
            FrontendMessage {
                id: None,
                data: Some(Data::ShellOutput(msg.data)),
            }
            .encode_to_vec(),
        );
    }
}
