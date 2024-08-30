use aes_gcm::{
    aead::{Aead, OsRng},
    AeadCore, Aes256Gcm, KeyInit, Nonce,
};
use derivative::Derivative;
use monitor_api::prost::Message as _;
use monitor_api::{
    ecies::{encrypt, PublicKey},
    Message,
};
use skynet_api::{actix_cloud::bail, Result};
use skynet_api::{
    actix_cloud::tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpStream,
    },
    anyhow,
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

pub struct Frame {
    pk: [u8; PUBLIC_KEY_SIZE],
    key: [u8; AES256_KEY_SIZE],
    stream: TcpStream,
    cipher: Aes256Gcm,
}

impl Frame {
    pub fn new(stream: TcpStream, pubkey: &PublicKey) -> Self {
        let key = Aes256Gcm::generate_key(OsRng);
        Self {
            pk: pubkey.serialize(),
            stream,
            cipher: Aes256Gcm::new(&key),
            key: key.into(),
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
        let len = self.stream.read_u32().await?;
        let mut buf = vec![0; len.try_into()?];
        self.stream.read_exact(&mut buf).await?;
        Ok(buf)
    }

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
