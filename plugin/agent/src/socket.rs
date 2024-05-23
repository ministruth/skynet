use anyhow::Result;
use derivative::Derivative;
use futures_util::{SinkExt, StreamExt};
use monitor_service::{
    client,
    server::{self, DataType},
};
use skynet::HyUuid;
use tokio::net::TcpStream;
use tokio_tungstenite::{tungstenite::Message, MaybeTlsStream, WebSocketStream};

#[derive(thiserror::Error, Derivative)]
#[derivative(Debug)]
pub enum SocketError {
    #[error("Connection lost")]
    ConnectionLost,

    #[error("Invalid message")]
    InvalidMessage,

    #[error("Invalid login response")]
    InvalidLoginResponse,
}

pub struct Socket(pub WebSocketStream<MaybeTlsStream<TcpStream>>);

impl Socket {
    pub async fn recv_msg(&mut self) -> Result<client::Message> {
        let msg = self.0.next().await.ok_or(SocketError::ConnectionLost)??;
        if let Message::Text(msg) = msg {
            Ok(serde_json::from_str::<client::Message>(&msg)?)
        } else {
            Err(SocketError::InvalidMessage.into())
        }
    }

    pub async fn send_msg_rsp(&mut self, id: &HyUuid, data: DataType) -> Result<()> {
        self.send_text(server::Message::new_rsp(id, data)).await
    }

    pub async fn send_msg(&mut self, data: DataType) -> Result<()> {
        self.send_text(server::Message::new(data)).await
    }

    pub async fn send_text(&mut self, msg: server::Message) -> Result<()> {
        self.0
            .send(Message::Text(msg.into()))
            .await
            .map_err(Into::into)
    }
}
