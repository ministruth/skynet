use std::net::SocketAddr;
use std::time::Duration;

use actix::clock::{interval, Instant, Interval};
use aes_gcm::aead::{Aead, OsRng};
use aes_gcm::{AeadCore, Aes256Gcm, KeyInit, Nonce};
use derivative::Derivative;
use miniz_oxide::deflate::compress_to_vec;
use monitor_api::{
    ecies::{self, SecretKey},
    message::Data,
    prost::Message as _,
    AgentStatus, HandshakeReqMessage, HandshakeRspMessage, HandshakeStatus, InfoMessage, Message,
    StatusReqMessage, StatusRspMessage, UpdateMessage, ID,
};
use skynet_api::actix_cloud::bail;
use skynet_api::actix_cloud::chrono::{DateTime, Utc};
use skynet_api::actix_cloud::tokio::io::{AsyncReadExt, AsyncWriteExt};
use skynet_api::actix_cloud::tokio::net::{TcpListener, TcpStream};
use skynet_api::actix_cloud::tokio::select;
use skynet_api::actix_cloud::tokio::sync::broadcast::{channel, Receiver, Sender};
use skynet_api::actix_cloud::tokio::sync::mpsc::{
    unbounded_channel, UnboundedReceiver, UnboundedSender,
};
use skynet_api::actix_cloud::tokio::time::sleep;
use skynet_api::request::Condition;
use skynet_api::{
    anyhow, async_trait,
    parking_lot::RwLock,
    sea_orm::TransactionTrait,
    tracing::{debug, error, field, info, info_span, warn, Instrument, Span},
    HyUuid, Result,
};

use crate::ws_handler::{ShellError, ShellOutput};
use crate::{AGENT_API, DB, RUNTIME, SERVICE, WEB_ADDRESS};

const AES256_KEY_SIZE: usize = 32;
const SECRET_KEY_SIZE: usize = 32;
const MAGIC_NUMBER: &[u8] = b"SKNT";

struct Frame {
    stream: TcpStream,
    cipher: Option<Aes256Gcm>,
    sk: [u8; SECRET_KEY_SIZE],
}

impl Frame {
    fn new(stream: TcpStream, sk: SecretKey) -> Self {
        Self {
            stream,
            cipher: None,
            sk: sk.serialize(),
        }
    }

    async fn close(&mut self) {
        let _ = self.stream.shutdown().await;
    }

    async fn send(&mut self, buf: &[u8]) -> Result<()> {
        let len = buf.len().try_into()?;
        self.stream.write_u32(len).await?;
        self.stream.write_all(buf).await?;
        self.stream.flush().await?;
        Ok(())
    }

    async fn send_msg(&mut self, msg: &Message) -> Result<()> {
        let mut buf = MAGIC_NUMBER.to_vec();
        buf.extend(msg.encode_to_vec());
        let nonce = Aes256Gcm::generate_nonce(&mut OsRng);
        let enc = self
            .cipher
            .as_ref()
            .unwrap()
            .encrypt(&nonce, buf.as_slice())
            .map_err(|e| anyhow!(e))?;
        let mut buf = nonce.to_vec();
        buf.extend(enc);
        self.send(&buf).await
    }

    async fn read(&mut self) -> Result<Vec<u8>> {
        let len = self.stream.read_u32().await?;
        let mut buf = vec![0; len.try_into()?];
        self.stream.read_exact(&mut buf).await?;
        Ok(buf)
    }

    async fn read_msg(&mut self) -> Result<Message> {
        let buf = self.read().await?;
        if let Some(cipher) = &self.cipher {
            let nonce = Nonce::from_slice(&buf[0..12]);
            let buf = cipher.decrypt(nonce, &buf[12..]).map_err(|e| anyhow!(e))?;
            if !buf.starts_with(MAGIC_NUMBER) {
                bail!("Invalid magic number");
            }
            Message::decode(&buf[MAGIC_NUMBER.len()..]).map_err(Into::into)
        } else {
            // handshake
            let data = ecies::decrypt(&self.sk, &buf)?;
            if data.len() > AES256_KEY_SIZE {
                let (key, uid) = data.split_at(AES256_KEY_SIZE);
                self.cipher = Some(Aes256Gcm::new_from_slice(key)?);
                Ok(Message {
                    seq: 0,
                    data: Some(Data::HandshakeReq(HandshakeReqMessage {
                        uid: String::from_utf8_lossy(uid).to_string(),
                    })),
                })
            } else {
                bail!("Invalid handshake data");
            }
        }
    }
}

struct Handler {
    shutdown_rx: Receiver<()>,
    client_seq: u64,
    server_seq: u64,
    trace_id: HyUuid,
    start_time: DateTime<Utc>,
    client_addr: SocketAddr,
    aid: Option<HyUuid>,
    status_clock: Option<Interval>,
    message: Option<UnboundedReceiver<Data>>,
}

impl Handler {
    fn new(trace_id: HyUuid, client_addr: SocketAddr, shutdown_rx: Receiver<()>) -> Self {
        Self {
            client_seq: 0,
            server_seq: 0,
            shutdown_rx,
            trace_id,
            start_time: Utc::now(),
            client_addr,
            aid: None,
            status_clock: None,
            message: None,
        }
    }

    fn new_server_msg(&mut self, data: Data) -> Message {
        let ret = Message {
            seq: self.server_seq,
            data: Some(data),
        };
        self.server_seq += 1;
        ret
    }

    async fn handshake(&mut self, frame: &mut Frame, msg: Message) -> Result<()> {
        if msg.seq == 0 && self.client_seq == 0 {
            if let Some(Data::HandshakeReq(data)) = msg.data {
                let tx = DB.get().unwrap().begin().await?;
                self.aid = Some(
                    if let Some(x) = SERVICE
                        .get()
                        .unwrap()
                        .login(&tx, data.uid, self.client_addr)
                        .await?
                    {
                        x
                    } else {
                        let _ = frame
                            .send_msg(&self.new_server_msg(Data::HandshakeRsp(
                                HandshakeRspMessage {
                                    status: HandshakeStatus::Logined.into(),
                                    trace_id: self.trace_id.to_string(),
                                },
                            )))
                            .await;
                        bail!("Already login");
                    },
                );
                tx.commit().await?;

                self.message = Some(SERVICE.get().unwrap().bind_message(&self.aid.unwrap()));
                Span::current().record("aid", self.aid.unwrap().to_string());
                self.start_time = Utc::now();
                info!(
                    _time = self.start_time.timestamp_micros(),
                    "Agent connection received"
                );
                return frame
                    .send_msg(
                        &self.new_server_msg(Data::HandshakeRsp(HandshakeRspMessage {
                            status: HandshakeStatus::Success.into(),
                            trace_id: self.trace_id.to_string(),
                        })),
                    )
                    .await;
            }
        }
        bail!("Invalid handshake message")
    }

    fn handle_status(&mut self, _frame: &mut Frame, data: StatusRspMessage) -> Result<()> {
        SERVICE
            .get()
            .unwrap()
            .update_status(&self.aid.unwrap(), data);
        Ok(())
    }

    async fn handle_info(&mut self, frame: &mut Frame, data: InfoMessage) -> Result<()> {
        let aid = self.aid.unwrap();
        let sys = data.os.clone().unwrap_or_default();
        let arch = data.arch.clone().unwrap_or_default();
        let version = data.version.clone();

        if data.report_rate != 0 {
            self.status_clock = Some(interval(Duration::from_secs(data.report_rate.into())));
        }

        let tx = DB.get().unwrap().begin().await?;
        SERVICE
            .get()
            .unwrap()
            .update(&tx, &self.aid.unwrap(), data)
            .await?;
        tx.commit().await?;
        if let Some(x) = AGENT_API.get() {
            if agent_api::Service::check_version(&version) {
                let sys = agent_api::System::parse(&sys);
                let arch = agent_api::Arch::parse(&arch);
                if sys.is_none() || arch.is_none() {
                    warn!(
                        arch = ?arch,
                        system = ?sys,
                        "Agent not update, platform invalid",
                    );
                }

                if let Some(data) = x.get_binary(sys.unwrap(), arch.unwrap()) {
                    if let Some(x) = SERVICE.get().unwrap().agent.write().get_mut(&aid) {
                        x.status = AgentStatus::Updating;
                    }
                    let crc = crc32fast::hash(&data);
                    let data = compress_to_vec(&data, 6);
                    frame
                        .send_msg(
                            &self.new_server_msg(Data::Update(UpdateMessage { data, crc32: crc })),
                        )
                        .await?;
                } else {
                    warn!(
                        file = %x.get_binary_name(sys.unwrap(), arch.unwrap()).to_string_lossy(),
                        "Agent not update, file not found",
                    );
                }
            }
        }
        Ok(())
    }

    async fn handle_msg(&mut self, frame: &mut Frame, msg: Message) -> Result<()> {
        if msg.seq < self.client_seq {
            debug!(
                seq = self.client_seq,
                msg_seq = msg.seq,
                "Invalid sequence number"
            );
            Ok(())
        } else {
            self.client_seq = msg.seq + 1;
            if let Some(data) = msg.data {
                match data {
                    Data::Info(data) => self.handle_info(frame, data).await,
                    Data::StatusRsp(data) => self.handle_status(frame, data),
                    Data::ShellOutput(mut data) => {
                        let id = HyUuid::parse(&data.token.unwrap_or_default())?;
                        data.token = None;
                        if let Some(addr) = WEB_ADDRESS.get().unwrap().read().get(&id) {
                            addr.do_send(ShellOutput { data });
                        }
                        Ok(())
                    }
                    Data::ShellError(mut data) => {
                        let id = HyUuid::parse(&data.token.unwrap_or_default())?;
                        data.token = None;
                        if let Some(addr) = WEB_ADDRESS.get().unwrap().read().get(&id) {
                            addr.do_send(ShellError { data });
                        }
                        Ok(())
                    }
                    _ => bail!("Invalid message type"),
                }
            } else {
                bail!("Invalid message")
            }
        }
    }

    async fn get_status_tick(c: &mut Option<Interval>) -> Option<Instant> {
        match c {
            Some(t) => Some(t.tick().await),
            None => None,
        }
    }

    async fn get_proxy_message(c: &mut Option<UnboundedReceiver<Data>>) -> Option<Data> {
        match c {
            Some(d) => d.recv().await,
            None => None,
        }
    }

    async fn send_status(&mut self, frame: &mut Frame) {
        let _ = frame
            .send_msg(&self.new_server_msg(Data::StatusReq(StatusReqMessage {
                time: Utc::now().timestamp_millis(),
            })))
            .await;
    }

    async fn process(&mut self, stream: TcpStream, key: SecretKey) {
        let mut frame = Frame::new(stream, key);
        loop {
            select! {
                msg = frame.read_msg() => {
                    match msg {
                        Ok(msg) => {
                            if self.aid.is_none() {
                                if let Err(e) = self.handshake(&mut frame, msg).await {
                                    debug!(error = %e, "Error handshake");
                                    frame.close().await;
                                }
                            } else if let Err(e) = self.handle_msg(&mut frame, msg).await {
                                debug!(error = %e, "Error handle message");
                            }
                        }
                        Err(e) => {
                            if self.aid.is_some() {
                                let end_time = Utc::now();
                                let time = (end_time - self.start_time).num_microseconds().unwrap_or(0);
                                warn!(_time = end_time.timestamp_micros(), alive_time = time, error = %e, "Connection lost");
                            }
                            break;
                        }
                    }
                }
                Some(_) = Self::get_status_tick(&mut self.status_clock) => {
                    self.send_status(&mut frame).await;
                }
                Some(data) = Self::get_proxy_message(&mut self.message) => {
                    if let Err(e) = frame.send_msg(&self.new_server_msg(data)).await {
                        debug!(error = %e, "Error send message");
                    }
                }
                _ = self.shutdown_rx.recv() =>{
                    if self.aid.is_some() {
                        let end_time = Utc::now();
                        let time = (end_time - self.start_time).num_microseconds().unwrap_or(0);
                        info!(_time = end_time.timestamp_micros(), alive_time = time, "Server shutdown");
                    }
                    break;
                }
            }
        }
        self.status_clock = None;
        if let Some(aid) = self.aid {
            SERVICE.get().unwrap().logout(&aid);
            self.message = None;
        }
    }
}

struct Listener {
    listener: TcpListener,
    passive: UnboundedReceiver<HyUuid>,
    shutdown_rx: Receiver<()>,
}

impl Listener {
    async fn new(
        addr: &str,
        passive: UnboundedReceiver<HyUuid>,
        shutdown_rx: Receiver<()>,
    ) -> Result<Self> {
        let listener = TcpListener::bind(&addr).await?;
        Ok(Self {
            listener,
            passive,
            shutdown_rx,
        })
    }

    async fn passive(addr: &str, key: SecretKey, rx: Receiver<()>) -> Result<()> {
        info!(plugin = %ID, "Monitor connecting to {}...", addr);
        let stream = TcpStream::connect(addr).await?;
        let addr = stream.peer_addr()?;
        let trace_id = HyUuid::new();
        Handler::new(trace_id, addr, rx)
            .process(stream, key)
            .instrument(info_span!("Agent connection", plugin = %ID, trace_id = %trace_id, ip = addr.to_string(), aid = field::Empty))
            .await;
        Ok(())
    }

    async fn passive_loop(key: SecretKey, rx: Receiver<()>, apid: HyUuid) -> Result<()> {
        loop {
            let tx = DB.get().unwrap().begin().await?;
            let m = SERVICE
                .get()
                .unwrap()
                .find_passive_by_id(&tx, &apid)
                .await?;
            tx.commit().await?;
            if let Some(m) = m {
                if let Err(e) = Self::passive(&m.address, key, rx.resubscribe()).await {
                    error!(plugin = %ID, error = %e, "Monitor connect error");
                }
                if m.retry_time != 0 {
                    sleep(Duration::from_secs(m.retry_time.try_into()?)).await;
                } else {
                    return Ok(());
                }
            } else {
                return Ok(());
            }
        }
    }

    async fn run(&mut self, key: SecretKey) {
        loop {
            select! {
                c = self.listener.accept() => {
                    match c {
                        Ok((stream, addr)) => {
                            let rx = self.shutdown_rx.resubscribe();
                            RUNTIME.get().unwrap().spawn(async move {
                                let trace_id = HyUuid::new();
                                Handler::new(trace_id, addr, rx)
                                    .process(stream, key)
                                    .instrument(info_span!("Agent connection", plugin = %ID, trace_id = %trace_id, ip = addr.to_string(), aid = field::Empty))
                                    .await;
                            });
                        }
                        Err(e) => debug!("{e}"),
                    }
                },
                Some(apid) = self.passive.recv() => {
                    let rx = self.shutdown_rx.resubscribe();
                    RUNTIME.get().unwrap().spawn(async move {
                        if let Err(e) =Self::passive_loop(key, rx, apid).await{
                            error!(plugin = %ID, error = %e, "Monitor passive agent error");
                        }
                    });
                }
                _ = self.shutdown_rx.recv() => {
                    return;
                }
            }
        }
    }
}

#[derive(Derivative)]
#[derivative(Default(new = "true"))]
pub struct Server {
    running: RwLock<bool>,
    passive: RwLock<Option<UnboundedSender<HyUuid>>>,
    shutdown_tx: RwLock<Option<Sender<()>>>,
}

#[async_trait]
impl monitor_api::Server for Server {
    async fn start(&self, addr: &str, key: SecretKey) -> Result<()> {
        let (tx, mut rx) = channel(1);
        let (passive_tx, passive_rx) = unbounded_channel();
        let mut listener = Listener::new(addr, passive_rx, tx.subscribe()).await?;
        *self.passive.write() = Some(passive_tx);
        *self.shutdown_tx.write() = Some(tx);
        *self.running.write() = true;

        let tx = DB.get().unwrap().begin().await?;
        let passive = SERVICE
            .get()
            .unwrap()
            .find_passive(&tx, Condition::new(Condition::all()))
            .await?
            .0;
        tx.commit().await?;
        for i in passive {
            self.connect(&i.id);
        }

        info!(plugin = %ID, "Monitor server listening on {addr}");
        select! {
            _ = listener.run(key) => {},
            _ = rx.recv() => {},
        }
        *self.running.write() = false;
        *self.shutdown_tx.write() = None;
        *self.passive.write() = None;
        info!(plugin = %ID, "Monitor server stopped");
        Ok(())
    }

    fn is_running(&self) -> bool {
        *self.running.read()
    }

    fn stop(&self) -> bool {
        self.shutdown_tx
            .read()
            .as_ref()
            .is_some_and(|x| x.send(()).is_ok())
    }

    fn connect(&self, apid: &HyUuid) -> bool {
        self.passive
            .read()
            .as_ref()
            .is_some_and(|x| x.send(*apid).is_ok())
    }
}
