use actix::{Actor, ActorContext, StreamHandler};
use actix_web_actors::ws::{start, Message, ProtocolError, WebsocketContext};
use derivative::Derivative;
use skynet::{
    actix_web::{
        web::{Data, Payload},
        Error, HttpRequest, HttpResponse,
    },
    anyhow::Result,
    request::Request,
    Skynet,
};

#[derive(Derivative)]
pub struct WSHandler {
    skynet: Data<Skynet>,
    request: Request,
}

impl Actor for WSHandler {
    type Context = WebsocketContext<Self>;
}

impl WSHandler {
    fn new(skynet: Data<Skynet>, request: Request) -> Self {
        Self { skynet, request }
    }
}

impl StreamHandler<Result<Message, ProtocolError>> for WSHandler {
    fn handle(&mut self, msg: Result<Message, ProtocolError>, ctx: &mut Self::Context) {
        match msg {
            Ok(Message::Ping(msg)) => ctx.pong(&msg),
            Ok(Message::Text(text)) => {
                // TODO
                println!("{text}");
            }
            Ok(Message::Close(reason)) => {
                ctx.close(reason);
                ctx.stop();
            }
            _ => ctx.stop(),
        }
    }
}

pub async fn service(
    req: HttpRequest,
    r: Request,
    skynet: Data<Skynet>,
    payload: Payload,
) -> Result<HttpResponse, Error> {
    start(WSHandler::new(skynet, r), &req, payload)
}
