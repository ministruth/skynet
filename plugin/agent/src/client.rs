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
use futures_util::{SinkExt, StreamExt};
use miniz_oxide::inflate::decompress_to_vec;
use monitor_service::{client, server};
use skynet::{
    tracing::{debug, error, info},
    utils, HyUuid,
};
use sysinfo::{CpuRefreshKind, Disks, MemoryRefreshKind, Networks, RefreshKind, System};
use tokio::{
    select,
    sync::mpsc::{self, UnboundedSender},
    time::sleep,
};
use tokio_tungstenite::{
    connect_async_with_config,
    tungstenite::{protocol::WebSocketConfig, Message},
};
use url::Url;

use crate::{
    get_uid,
    shell::ShellInstance,
    socket::{Socket, SocketError},
};

struct ConnectionState<'a> {
    shell: HashMap<String, ShellInstance>,
    stat_sys: System,
    stat_disk: Disks,
    stat_network: Networks,
    uid: String,
    disk: &'a [String],
    interface: &'a [String],
    output: Option<UnboundedSender<server::Message>>,
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
    ws: &mut Socket,
    state: &mut ConnectionState<'a>,
    msg_id: HyUuid,
    data: client::ShellConnect,
) -> Result<()> {
    let token = utils::rand_string(32);
    let inst = ShellInstance::new(
        &token,
        &data.cmd,
        data.rows,
        data.cols,
        state.output.clone(),
    );
    match inst {
        Ok(inst) => {
            state.shell.insert(token.clone(), inst);
            ws.send_msg_rsp(
                &msg_id,
                server::DataType::ShellConnect(server::ShellConnect {
                    token: token.clone(),
                    error: String::new(),
                }),
            )
            .await?;
            info!(_token = token, "New shell connected");
            Ok(())
        }
        Err(e) => {
            ws.send_msg_rsp(
                &msg_id,
                server::DataType::ShellConnect(server::ShellConnect {
                    token: String::new(),
                    error: e.to_string(),
                }),
            )
            .await?;
            Err(e)
        }
    }
}

#[allow(clippy::needless_pass_by_value)]
fn resize_handler(state: &ConnectionState, data: client::ShellResize) -> Result<()> {
    match state.shell.get(&data.token) {
        Some(x) => x.resize(data.rows, data.cols),
        None => bail!("Shell token not found"),
    }
}

fn input_handler(state: &mut ConnectionState, data: client::ShellInput) -> Result<()> {
    match state.shell.get_mut(&data.token) {
        Some(x) => x.write(&STANDARD.decode(data.data)?),
        None => bail!("Shell token not found"),
    }
}

async fn update_handler(ws: &mut Socket, data: client::Update) -> Result<()> {
    let exe = env::current_exe()?;
    let file = STANDARD.decode(data.data)?;
    let file = decompress_to_vec(&file).map_err(|e| anyhow::anyhow!(e.to_string()))?;
    let crc = crc32fast::hash(&file);
    if crc == data.crc32 {
        let new_path = format!("_agent_update{}", consts::EXE_SUFFIX);
        fs::write(&new_path, file)?;
        self_replace::self_replace(&new_path)?;
        fs::remove_file(new_path)?;
        let _ = ws.0.close(None).await;
        info!(crc32 = crc, "Trigger update");
        return Command::new(exe)
            .args(env::args().skip(1))
            .spawn()
            .map_err(Into::into)
            .and(Ok(()));
    }
    bail!("Update: CRC32 mismatch");
}

async fn status_handler<'a>(
    ws: &mut Socket,
    state: &mut ConnectionState<'a>,
    msg_id: HyUuid,
    data: client::Status,
) {
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
    ws.send_msg_rsp(
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
    .await
    .unwrap_or_else(|e| debug!(error = %e, "Send status error"));
}

async fn login_handler(
    ws: &mut Socket,
    msg_id: HyUuid,
    login_id: HyUuid,
    data: client::Login,
) -> Result<()> {
    if msg_id != login_id {
        bail!("Invalid login response");
    } else if data.code != 0 {
        bail!("Login error: code `{}`: {}", data.code, data.msg);
    } else {
        info!("Login success");

        ws.send_msg(server::DataType::Info(server::Info {
            version: env!("CARGO_PKG_VERSION").to_owned(),
            os: System::name(),
            system: System::long_os_version(),
            arch: consts::ARCH.to_owned(),
            hostname: System::host_name(),
        }))
        .await
        .unwrap_or_else(|e| debug!(error=%e, "Send info error"));
        Ok(())
    }
}

#[allow(clippy::needless_pass_by_value)]
fn disconnect_handler(state: &mut ConnectionState, data: client::ShellDisconnect) {
    if let Some(x) = state.shell.remove(&data.token) {
        drop(x); // explicitly kill
    }
    info!(_token = data.token, "Shell disconnected");
}

async fn handler<'a>(
    ws: &mut Socket,
    state: &mut ConnectionState<'a>,
    msg: String,
) -> Result<Option<Result<()>>> {
    match serde_json::from_str::<client::Message>(&msg) {
        Ok(msg) => match msg.data {
            client::DataType::Update(data) => update_handler(ws, data).await?,
            client::DataType::Status(data) => status_handler(ws, state, msg.id, data).await,
            client::DataType::ShellConnect(data) => shell_handler(ws, state, msg.id, data).await?,
            client::DataType::ShellResize(data) => resize_handler(state, data)?,
            client::DataType::ShellInput(data) => input_handler(state, data)?,
            client::DataType::ShellDisconnect(data) => disconnect_handler(state, data),
            client::DataType::Reconnect => {
                return Ok(Some(Err(anyhow!("Receive reconnect signal from server"))))
            }
            client::DataType::Quit => {
                info!("Receive quit signal from server");
                return Ok(Some(Ok(())));
            }
            _ => debug!(msg = ?msg, "Invalid message"),
        },
        Err(e) => debug!(error = %e,"Parse message error"),
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
    let (ws, _) = connect_async_with_config(
        url,
        Some(WebSocketConfig {
            max_frame_size: Some(1024 * 1024 * 512),
            max_message_size: Some(1024 * 1024 * 512),
            ..Default::default()
        }),
        false,
    )
    .await?;
    let mut ws = Socket(ws);
    info!("Connected");

    // login
    state.uid = get_uid();
    let login_msg = server::Message::new(server::DataType::Login(server::Login {
        uid: state.uid.clone(),
        token: token.to_owned(),
    }));
    let login_id = login_msg.id;
    ws.send_text(login_msg).await?;

    let msg = ws.recv_msg().await?;
    if let client::DataType::Login(data) = msg.data {
        login_handler(&mut ws, msg.id, login_id, data).await?;
        *wait_time = 1;
    } else {
        bail!(SocketError::InvalidLoginResponse)
    }

    let (shell_tx, mut shell_rx) = mpsc::unbounded_channel();
    state.output = Some(shell_tx);
    loop {
        select! {
            msg = shell_rx.recv() => {
                if let Some(data) = msg {
                    ws.send_text(data)
                    .await
                    .unwrap_or_else(|e| debug!(error = %e, "Send shell output error"));
                }
            },
            msg = ws.0.next() => {
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
                                let _ = ws.0.send(Message::Pong(x)).await;
                            }
                            Message::Close(x) => {
                                let _ = ws.0.close(x).await;
                                break;
                            }
                            _ => debug!("Unknown message"),
                        },
                        Err(e) => error!("{e}"),
                    }
                } else {
                    break;
                }
            }
        }
    }
    bail!(SocketError::ConnectionLost)
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
