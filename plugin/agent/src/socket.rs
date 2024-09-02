use std::io;

use aes_gcm::{
    aead::{Aead, OsRng},
    AeadCore, Aes256Gcm, KeyInit, Nonce,
};
use bytes::BytesMut;
use derivative::Derivative;
use skynet_api::{
    actix_cloud::tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpStream,
    },
    anyhow,
};
use skynet_api::{
    actix_cloud::{bail, tokio::io::AsyncRead},
    Result,
};
use skynet_api_monitor::prost::Message as _;
use skynet_api_monitor::{
    ecies::{encrypt, PublicKey},
    Message,
};

const NONCE_SIZE: usize = 12;
const PUBLIC_KEY_SIZE: usize = 65;
const AES256_KEY_SIZE: usize = 32;
const MAGIC_NUMBER: &[u8] = b"SKNT";

#[derive(thiserror::Error, Derivative)]
#[derivative(Debug)]
pub enum SocketError {
    #[error("Invalid message")]
    InvalidMessage,

    #[error("Invalid magic number")]
    InvalidMagicNumber,

    #[error("Receive reconnect signal from server")]
    Reconnect,

    #[error("Already login")]
    AlreadyLogin,

    #[error("Shell feature is disabled")]
    ShellDisabled,
}

#[derive(Derivative)]
#[derivative(Default(new = "true"))]
struct FrameLen {
    data: [u8; 4],
    len: usize,
}

impl FrameLen {
    async fn next<R>(&mut self, io: &mut R) -> Result<u32>
    where
        R: AsyncRead + Unpin,
    {
        while self.len < 4 {
            let cnt = io.read(&mut self.data[self.len..]).await?;
            if cnt == 0 {
                self.len = 0;
                return Err(io::Error::from(io::ErrorKind::UnexpectedEof).into());
            }
            self.len += cnt;
        }
        Ok(u32::from_be_bytes(self.data))
    }

    fn reset(&mut self) {
        self.len = 0;
    }
}

pub struct Frame {
    pk: [u8; PUBLIC_KEY_SIZE],
    key: [u8; AES256_KEY_SIZE],
    stream: TcpStream,
    cipher: Aes256Gcm,
    len: FrameLen,
}

impl Frame {
    pub fn new(stream: TcpStream, pubkey: &PublicKey) -> Self {
        let key = Aes256Gcm::generate_key(OsRng);
        Self {
            pk: pubkey.serialize(),
            stream,
            cipher: Aes256Gcm::new(&key),
            key: key.into(),
            len: FrameLen::new(),
        }
    }

    pub async fn close(&mut self) {
        let _ = self.stream.shutdown().await;
    }

    pub async fn handshake(&mut self, uid: &str) -> Result<()> {
        let mut buf = self.key.to_vec();
        buf.extend(uid.as_bytes());
        let msg = encrypt(&self.pk, &buf)?;
        self.send(&msg).await
    }

    pub async fn send(&mut self, buf: &[u8]) -> Result<()> {
        let len = buf.len().try_into()?;
        self.stream.write_u32(len).await?;
        self.stream.write_all(buf).await?;
        self.stream.flush().await?;
        Ok(())
    }

    pub async fn send_msg(&mut self, msg: &Message) -> Result<()> {
        let mut buf = MAGIC_NUMBER.to_vec();
        buf.extend(msg.encode_to_vec());
        let nonce = Aes256Gcm::generate_nonce(&mut OsRng);
        let enc = self
            .cipher
            .encrypt(&nonce, buf.as_slice())
            .map_err(|e| anyhow!(e))?;
        let mut buf = nonce.to_vec();
        buf.extend(enc);
        self.send(&buf).await
    }

    pub async fn read(&mut self) -> Result<Vec<u8>> {
        let len = self.len.next(&mut self.stream).await?;
        let mut ret = BytesMut::with_capacity(len.try_into()?);
        if self.stream.read_buf(&mut ret).await? == 0 {
            self.len.reset();
            return Err(io::Error::from(io::ErrorKind::UnexpectedEof).into());
        }
        self.len.reset();
        Ok(ret.into())
    }

    /// Read message from frame.
    ///
    /// # Cancel safety
    /// This function is cancellation safe.
    pub async fn read_msg(&mut self) -> Result<Message> {
        let buf = self.read().await?;
        let nonce = Nonce::from_slice(&buf[0..NONCE_SIZE]);
        let buf = self
            .cipher
            .decrypt(nonce, &buf[NONCE_SIZE..])
            .map_err(|e| anyhow!(e))?;
        if !buf.starts_with(MAGIC_NUMBER) {
            bail!(SocketError::InvalidMagicNumber);
        }
        Message::decode(&buf[MAGIC_NUMBER.len()..]).map_err(Into::into)
    }
}
