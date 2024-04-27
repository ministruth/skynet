use std::{
    cmp::min,
    collections::{HashMap, HashSet},
    env::{self, consts},
    fs,
    process::Command,
    time::Duration,
};

use anyhow::{anyhow, bail, Result};
use base64::{engine::general_purpose::STANDARD, Engine};
use derivative::Derivative;
use futures_util::{SinkExt, StreamExt};
use log::{debug, error, info};
use miniz_oxide::inflate::decompress_to_vec;
use monitor_service::{client, server};
use skynet::{utils, HyUuid};
use sysinfo::{CpuRefreshKind, Disks, MemoryRefreshKind, Networks, RefreshKind, System};
use tokio::{
    net::TcpStream,
    select,
    sync::mpsc::{self, UnboundedSender},
    time::sleep,
};
use tokio_tungstenite::{
    connect_async_with_config,
    tungstenite::{protocol::WebSocketConfig, Message},
    MaybeTlsStream, WebSocketStream,
};
use url::Url;

use crate::{get_uid, shell::ShellInstance};

#[derive(thiserror::Error, Derivative)]
#[derivative(Debug)]
pub enum AgentError {
    #[error("Connection lost")]
    ConnectionLost,

    #[error("Invalid login response")]
    InvalidLoginResponse,
}

struct ConnectionState<'a> {
    shell: HashMap<String, ShellInstance>,
    stat_sys: System,
    stat_disk: Disks,
    stat_network: Networks,
    uid: String,
    disk: &'a [String],
    interface: &'a [String],
    output: Option<UnboundedSender<Vec<u8>>>,
}

impl<'a> ConnectionState<'a> {
    fn new(disk: &'a [String], interface: &'a [String]) -> Self {
        Self {
            shell: HashMap::new(),
            stat_sys: System::new(),
            stat_disk: Disks::new_with_refreshed_list(),
            stat_network: Networks::new_with_refreshed_list(),
            uid: String::new(),
            output: None,
            disk,
            interface,
        }
    }
}

async fn shell_handler<'a>(
    ws: &mut WebSocketStream<MaybeTlsStream<TcpStream>>,
    state: &mut ConnectionState<'a>,
    msg_id: HyUuid,
    data: client::ShellConnect,
) -> Result<()> {
    let inst = ShellInstance::new(&data.cmd, data.rows, data.cols, state.output.clone())?;
    let token = utils::rand_string(32);
    state.shell.insert(token.clone(), inst);
    ws.send(Message::Text(
        server::Message::new_rsp(
            &msg_id,
            server::DataType::ShellConnect(server::ShellConnect { token }),
        )
        .into(),
    ))
    .await?;
    Ok(())
}

#[allow(clippy::needless_pass_by_value)]
fn resize_handler(
    _ws: &WebSocketStream<MaybeTlsStream<TcpStream>>,
    state: &ConnectionState,
    _msg_id: HyUuid,
    data: client::ShellResize,
) -> Result<()> {
    match state.shell.get(&data.token) {
        Some(x) => x.resize(data.rows, data.cols),
        None => bail!("Shell token not found"),
    }
}

fn input_handler(
    _ws: &WebSocketStream<MaybeTlsStream<TcpStream>>,
    state: &mut ConnectionState,
    _msg_id: HyUuid,
    data: client::ShellInput,
) -> Result<()> {
    match state.shell.get_mut(&data.token) {
        Some(x) => x.write(&STANDARD.decode(data.data)?),
        None => bail!("Shell token not found"),
    }
}

async fn update_handler<'a>(
    ws: &mut WebSocketStream<MaybeTlsStream<TcpStream>>,
    _state: &ConnectionState<'a>,
    _msg_id: HyUuid,
    data: client::Update,
) -> Result<()> {
    let exe = env::current_exe()?;
    let file = STANDARD.decode(data.data)?;
    let file = decompress_to_vec(&file).map_err(|e| anyhow::anyhow!(e.to_string()))?;
    let crc = crc32fast::hash(&file);
    if crc == data.crc32 {
        let new_path = format!("_agent_update{}", consts::EXE_SUFFIX);
        fs::write(&new_path, file)?;
        self_replace::self_replace(&new_path)?;
        fs::remove_file(new_path)?;
        let _ = ws.close(None).await;
        info!("Trigger update, crc32: {}", crc);
        return Command::new(exe)
            .args(env::args().skip(1))
            .spawn()
            .map_err(Into::into)
            .and(Ok(()));
    }
    bail!("Update: CRC32 mismatch");
}

async fn status_handler<'a>(
    ws: &mut WebSocketStream<MaybeTlsStream<TcpStream>>,
    state: &mut ConnectionState<'a>,
    msg_id: HyUuid,
    data: client::Status,
) -> Result<()> {
    state.stat_sys.refresh_specifics(
        RefreshKind::new()
            .with_cpu(CpuRefreshKind::everything())
            .with_memory(MemoryRefreshKind::everything()),
    );
    state.stat_disk.refresh();
    state.stat_network.refresh();
    let mut disk_byte = 0;
    let mut total_disk_byte = 0;
    let mut visited_disk = HashSet::new(); // disk name may be duplicated.
    for i in &state.stat_disk {
        let name = i.name().to_string_lossy().to_string();
        if state.disk.contains(&name) && visited_disk.insert(name) {
            disk_byte += i.total_space() - i.available_space();
            total_disk_byte += i.total_space();
        }
    }
    let mut band_up = 0;
    let mut band_down = 0;
    for i in &state.stat_network {
        if state.interface.contains(i.0) {
            band_up += i.1.total_transmitted();
            band_down += i.1.total_received();
        }
    }
    ws.send(Message::Text(
        server::Message::new_rsp(
            &msg_id,
            server::DataType::Status(server::Status {
                time: data.time,
                cpu: state.stat_sys.global_cpu_info().cpu_usage(),
                memory: state.stat_sys.used_memory(),
                total_memory: state.stat_sys.total_memory(),
                disk: disk_byte,
                total_disk: total_disk_byte,
                band_up,
                band_down,
            }),
        )
        .into(),
    ))
    .await
    .unwrap_or_else(|e| debug!("Send status error: {e}"));
    Ok(())
}

async fn login_handler(
    ws: &mut WebSocketStream<MaybeTlsStream<TcpStream>>,
    msg_id: HyUuid,
    login_id: HyUuid,
    data: client::Login,
    wait_time: &mut u32,
) -> Result<()> {
    if msg_id != login_id {
        bail!("Invalid login response");
    } else if data.code != 0 {
        bail!("Login error: code `{}`: {}", data.code, data.msg);
    } else {
        *wait_time = 1;
        info!("Login success");

        ws.send(Message::Text(
            server::Message::new(server::DataType::Info(server::Info {
                version: env!("CARGO_PKG_VERSION").to_owned(),
                os: System::name(),
                system: System::long_os_version(),
                arch: consts::ARCH.to_owned(),
                hostname: System::host_name(),
            }))
            .into(),
        ))
        .await
        .unwrap_or_else(|e| debug!("Send info error: {e}"));
        Ok(())
    }
}

async fn handler<'a>(
    ws: &mut WebSocketStream<MaybeTlsStream<TcpStream>>,
    state: &mut ConnectionState<'a>,
    msg: String,
) -> Result<Option<Result<()>>> {
    match serde_json::from_str::<client::Message>(&msg) {
        Ok(msg) => match msg.data {
            client::DataType::Update(data) => update_handler(ws, state, msg.id, data).await?,
            client::DataType::Status(data) => status_handler(ws, state, msg.id, data).await?,
            client::DataType::ShellConnect(data) => shell_handler(ws, state, msg.id, data).await?,
            client::DataType::ShellResize(data) => resize_handler(ws, state, msg.id, data)?,
            client::DataType::ShellInput(data) => input_handler(ws, state, msg.id, data)?,
            client::DataType::Reconnect => {
                return Ok(Some(Err(anyhow!("Receive reconnect signal from server"))))
            }
            client::DataType::Quit => {
                info!("Receive quit signal from server");
                return Ok(Some(Ok(())));
            }
            _ => debug!("Invalid message type"),
        },
        Err(e) => debug!("Parse message error: {e}"),
    }
    Ok(None)
}

async fn connect<'a>(
    addr: &str,
    token: &str,
    mut state: ConnectionState<'a>,
    wait_time: &mut u32,
) -> Result<()> {
    let url = Url::parse(&format!(
        "{addr}/api/plugins/{}/agents/ws",
        monitor_service::ID
    ))
    .unwrap();
    info!("Connecting to {url}");
    let (mut ws, _) = connect_async_with_config(
        url,
        Some(WebSocketConfig {
            max_frame_size: Some(1024 * 1024 * 512),
            max_message_size: Some(1024 * 1024 * 512),
            ..Default::default()
        }),
        false,
    )
    .await?;
    info!("Connected");

    // login
    state.uid = get_uid();
    let login_msg = server::Message::new(server::DataType::Login(server::Login {
        uid: state.uid.clone(),
        token: token.to_owned(),
    }));
    ws.send(Message::Text(login_msg.json())).await?;

    let msg = ws.next().await.ok_or(AgentError::ConnectionLost)??;
    if let Message::Text(msg) = msg {
        let msg = serde_json::from_str::<client::Message>(&msg)?;
        if let client::DataType::Login(data) = msg.data {
            login_handler(&mut ws, msg.id, login_msg.id, data, wait_time).await?;
        } else {
            bail!(AgentError::InvalidLoginResponse)
        }
    } else {
        bail!(AgentError::InvalidLoginResponse)
    }

    let (shell_tx, mut shell_rx) = mpsc::unbounded_channel();
    state.output = Some(shell_tx);
    loop {
        select! {
            msg = shell_rx.recv() => {
                if let Some(data) = msg {
                    ws.send(Message::Text(
                        server::Message::new(server::DataType::ShellOutput(server::ShellOutput {
                            data: STANDARD.encode(data),
                        }))
                        .into(),
                    ))
                    .await
                    .unwrap_or_else(|e| debug!("Send shell output error: {e}"));
                }
            },
            msg = ws.next() => {
                if let Some(msg) = msg {
                    match msg {
                        Ok(msg) => match msg {
                            Message::Text(msg) => match handler(&mut ws, &mut state, msg).await {
                                Ok(ret) => {
                                    if let Some(ret) = ret {
                                        return ret;
                                    }
                                }
                                Err(e) => error!("{e}"),
                            },
                            Message::Ping(x) => {
                                let _ = ws.send(Message::Pong(x)).await;
                            }
                            Message::Close(x) => {
                                let _ = ws.close(x).await;
                                break;
                            }
                            _ => debug!("WS: Unknown message"),
                        },
                        Err(e) => error!("WS: {e}"),
                    }
                } else {
                    break;
                }
            }
        }
    }
    bail!(AgentError::ConnectionLost)
}

#[allow(clippy::while_let_loop)]
pub async fn run(
    addr: String,
    token: String,
    max_time: u32,
    disk: Vec<String>,
    interface: Vec<String>,
) {
    let mut wait_time = 1;
    loop {
        let state = ConnectionState::new(&disk, &interface);
        if let Err(e) = connect(&addr, &token, state, &mut wait_time).await {
            error!("{e}");
            info!("Wait for {wait_time} seconds to reconnect");
            sleep(Duration::from_secs(wait_time.into())).await;
            wait_time = min(max_time, wait_time * 2);
        } else {
            break;
        }
    }
}
