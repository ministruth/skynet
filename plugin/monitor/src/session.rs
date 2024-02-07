use actix::{ActorContext, Handler, Message};
use monitor_service::client;

use crate::ws::WSHandler;

#[derive(Message)]
#[rtype(result = "()")]
pub struct CloseConnection;

impl Handler<CloseConnection> for WSHandler {
    type Result = ();

    fn handle(&mut self, _: CloseConnection, ctx: &mut Self::Context) -> Self::Result {
        ctx.text(client::Message::new(client::DataType::Quit));
        ctx.close(None);
        ctx.stop();
    }
}
