use std::{
    cmp::min,
    collections::{HashMap, HashSet},
    env::{self, consts},
    fs::{self, create_dir_all},
    net::SocketAddr,
    process::Command,
    sync::atomic::{AtomicU32, Ordering},
    time::Duration,
};

use miniz_oxide::inflate::decompress_to_vec;
use path_clean::clean;
use skynet_api::{
    actix_cloud::{
        bail,
        tokio::{
            net::{TcpListener, TcpStream},
            select, spawn,
            sync::mpsc::{self, unbounded_channel, UnboundedSender},
            time::sleep,
        },
    },
    anyhow,
};
use skynet_api::{
    tracing::{debug, error, info, info_span, warn, Instrument},
    HyUuid, Result,
};
use skynet_api_monitor::{
    ecies::PublicKey, message::Data, CommandKillMessage, CommandReqMessage, FileReqMessage,
    FileRspMessage, HandshakeStatus, InfoMessage, Message, ShellConnectMessage,
    ShellDisconnectMessage, ShellErrorMessage, ShellInputMessage, ShellResizeMessage,
    StatusReqMessage, StatusRspMessage, UpdateMessage,
};
use sysinfo::{CpuRefreshKind, Disks, MemoryRefreshKind, Networks, RefreshKind, System};

use crate::{
    command::CommandInstance,
    get_uid,
    shell::ShellInstance,
    socket::{Frame, SocketError},
    RunArgs,
};

static WAIT_TIME: AtomicU32 = AtomicU32::new(1);

fn round_u16(v: u32) -> u16 {
    min(v, u16::MAX.into()) as u16
}

struct ConnectionState {
    client_seq: u64,
    server_seq: u64,
    trace_id: HyUuid,

    command: HashMap<HyUuid, CommandInstance>,
    shell: HashMap<HyUuid, ShellInstance>,
    stat_sys: System,
    stat_disk: Disks,
    stat_network: Networks,
    uid: String,
    output: Option<UnboundedSender<Data>>,
    args: RunArgs,
}

impl ConnectionState {
    fn new(args: RunArgs) -> Self {
        Self {
            client_seq: 0,
            server_seq: 0,
            trace_id: HyUuid::nil(),
            command: HashMap::new(),
            shell: HashMap::new(),
            stat_sys: System::new(),
            stat_disk: Disks::new_with_refreshed_list(),
            stat_network: Networks::new_with_refreshed_list(),
            uid: get_uid(),
            output: None,
            args,
        }
    }

    fn new_client_msg(&mut self, data: Data) -> Message {
        let ret = Message {
            seq: self.client_seq,
            data: Some(data),
        };
        self.client_seq += 1;
        ret
    }
}

async fn shell_handler(
    frame: &mut Frame,
    state: &mut ConnectionState,
    data: ShellConnectMessage,
) -> Result<Option<Result<()>>> {
    let id = HyUuid::parse(&data.token)?;
    let inst = ShellInstance::new(
        &data.token,
        &data.cmd,
        round_u16(data.rows),
        round_u16(data.cols),
        state.output.clone(),
    );
    match inst {
        Ok(inst) => {
            state.shell.insert(id, inst);
            info!(token = data.token, "New shell connected");
            Ok(None)
        }
        Err(e) => {
            frame
                .send_msg(&state.new_client_msg(Data::ShellError(ShellErrorMessage {
                    token: Some(data.token),
                    error: e.to_string(),
                })))
                .await?;
            Err(e)
        }
    }
}

fn resize_handler(
    _frame: &mut Frame,
    state: &mut ConnectionState,
    data: ShellResizeMessage,
) -> Result<Option<Result<()>>> {
    match state
        .shell
        .get(&HyUuid::parse(&data.token.unwrap_or_default())?)
    {
        Some(x) => x
            .resize(round_u16(data.rows), round_u16(data.cols))
            .map(|_| None),
        None => bail!("Shell token not found"),
    }
}

fn input_handler(
    _frame: &mut Frame,
    state: &mut ConnectionState,
    data: ShellInputMessage,
) -> Result<Option<Result<()>>> {
    match state
        .shell
        .get_mut(&HyUuid::parse(&data.token.unwrap_or_default())?)
    {
        Some(x) => x.write(&data.data).map(|_| None),
        None => bail!("Shell token not found"),
    }
}

async fn update_handler(
    frame: &mut Frame,
    state: &mut ConnectionState,
    data: UpdateMessage,
) -> Result<Option<Result<()>>> {
    let exe = env::current_exe()?;
    let file = decompress_to_vec(&data.data).map_err(|e| anyhow!(e.to_string()))?;
    let crc = crc32fast::hash(&file);
    if crc == data.crc32 {
        let new_path = format!("_agent_update{}", consts::EXE_SUFFIX);
        fs::write(&new_path, file)?;
        self_replace::self_replace(&new_path)?;
        fs::remove_file(new_path)?;
        let _ = frame.close().await;
        info!(crc32 = crc, "Trigger update");
        if state.args.restart {
            return Command::new(exe)
                .args(env::args().skip(1))
                .spawn()
                .map_err(Into::into)
                .and(Ok(Some(Ok(()))));
        } else {
            return Ok(Some(Ok(())));
        }
    }
    bail!("Update: CRC32 mismatch");
}

async fn status_handler(
    frame: &mut Frame,
    state: &mut ConnectionState,
    data: StatusReqMessage,
) -> Result<Option<Result<()>>> {
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
        if state.args.disk.contains(&name) && visited_disk.insert(name) {
            disk_byte += i.total_space() - i.available_space();
            total_disk_byte += i.total_space();
        }
    }
    let mut band_up = 0;
    let mut band_down = 0;
    for i in &state.stat_network {
        if state.args.interface.contains(i.0) {
            band_up += i.1.total_transmitted();
            band_down += i.1.total_received();
        }
    }
    frame
        .send_msg(&state.new_client_msg(Data::StatusRsp(StatusRspMessage {
            time: data.time,
            cpu: state.stat_sys.global_cpu_usage(),
            memory: state.stat_sys.used_memory(),
            total_memory: state.stat_sys.total_memory(),
            disk: disk_byte,
            total_disk: total_disk_byte,
            band_up,
            band_down,
        })))
        .await?;
    Ok(None)
}

fn disconnect_handler(
    _frame: &mut Frame,
    state: &mut ConnectionState,
    data: ShellDisconnectMessage,
) -> Result<Option<Result<()>>> {
    let token = HyUuid::parse(&data.token.unwrap_or_default())?;
    if let Some(x) = state.shell.remove(&token) {
        drop(x); // explicitly kill
        info!(%token, "Shell disconnected");
    }
    Ok(None)
}

async fn file_handler(
    frame: &mut Frame,
    state: &mut ConnectionState,
    data: FileReqMessage,
) -> Result<Option<Result<()>>> {
    let ret = (|| {
        let path = clean(state.args.data.join(&data.path));
        let save_folder = clean(&state.args.data);
        let mut it = path.ancestors();
        it.next();
        let mut flag = false;
        for i in it {
            if i == save_folder {
                flag = true;
                break;
            }
        }
        if !flag {
            bail!("File path invalid");
        }
        create_dir_all(path.parent().unwrap())?;
        let file = decompress_to_vec(&data.data).map_err(|e| anyhow!(e.to_string()))?;
        fs::write(path, file)?;
        Ok(None)
    })();
    if let Err(e) = &ret {
        frame
            .send_msg(&state.new_client_msg(Data::FileRsp(FileRspMessage {
                id: data.id,
                code: 1,
                message: e.to_string(),
            })))
            .await?;
    } else {
        info!(id = data.id, path = data.path, "New file transferred");
        frame
            .send_msg(&state.new_client_msg(Data::FileRsp(FileRspMessage {
                id: data.id,
                code: 0,
                message: String::from("Success"),
            })))
            .await?;
    }
    ret
}

fn command_handler(
    _frame: &mut Frame,
    state: &mut ConnectionState,
    data: CommandReqMessage,
) -> Result<Option<Result<()>>> {
    let id = HyUuid::parse(&data.id)?;
    let inst = CommandInstance::new(&data.id, &data.cmd, state.output.clone())?;
    state.command.insert(id, inst);
    info!(id = data.id, "New command executed");
    Ok(None)
}

fn kill_handler(
    _frame: &mut Frame,
    state: &mut ConnectionState,
    data: CommandKillMessage,
) -> Result<Option<Result<()>>> {
    match state.command.get(&HyUuid::parse(&data.id)?) {
        Some(x) => x.kill(data.force).map(|_| None),
        None => bail!("Command cid not found"),
    }
}

async fn handle_msg(
    frame: &mut Frame,
    state: &mut ConnectionState,
    msg: Message,
) -> Result<Option<Result<()>>> {
    if msg.seq < state.server_seq {
        debug!(
            seq = state.server_seq,
            msg_seq = msg.seq,
            "Invalid sequence number"
        );
        Ok(None)
    } else {
        state.server_seq = msg.seq;
        if let Some(data) = msg.data {
            match data {
                Data::Reconnect(_) => Ok(Some(Err(anyhow!(SocketError::Reconnect)))),
                Data::Quit(_) => {
                    info!("Receive quit signal from server");
                    Ok(Some(Ok(())))
                }
                Data::StatusReq(data) => status_handler(frame, state, data).await,
                Data::Update(data) => update_handler(frame, state, data).await,
                Data::ShellConnect(data) => {
                    if state.args.disable_shell {
                        bail!(SocketError::ShellDisabled)
                    } else {
                        shell_handler(frame, state, data).await
                    }
                }
                Data::ShellInput(data) => {
                    if state.args.disable_shell {
                        bail!(SocketError::ShellDisabled)
                    } else {
                        input_handler(frame, state, data)
                    }
                }
                Data::ShellResize(data) => {
                    if state.args.disable_shell {
                        bail!(SocketError::ShellDisabled)
                    } else {
                        resize_handler(frame, state, data)
                    }
                }
                Data::ShellDisconnect(data) => {
                    if state.args.disable_shell {
                        bail!(SocketError::ShellDisabled)
                    } else {
                        disconnect_handler(frame, state, data)
                    }
                }
                Data::FileReq(data) => file_handler(frame, state, data).await,
                Data::CommandReq(data) => command_handler(frame, state, data),
                Data::CommandKill(data) => kill_handler(frame, state, data),
                _ => bail!(SocketError::InvalidMessage),
            }
        } else {
            bail!(SocketError::InvalidMessage)
        }
    }
}

async fn active(addr: &str, pubkey: &PublicKey, state: ConnectionState) -> Result<()> {
    info!("Connecting to {addr}...");
    let stream = TcpStream::connect(addr).await?;
    let addr = stream.peer_addr()?;
    connect(stream, addr, pubkey, state).await
}

async fn connect(
    stream: TcpStream,
    addr: SocketAddr,
    pubkey: &PublicKey,
    mut state: ConnectionState,
) -> Result<()> {
    info!("Handshaking...");
    let mut frame = Frame::new(stream, pubkey);
    frame.handshake(&state.uid).await?;
    state.client_seq = 1;

    let msg = frame.read_msg().await?;
    if msg.seq == 0 && state.server_seq == 0 {
        if let Some(Data::HandshakeRsp(data)) = msg.data {
            if data.status == HandshakeStatus::Success as i32 {
                info!(trace_id = data.trace_id, "Connected");
                state.trace_id = HyUuid::parse(&data.trace_id)?;
                state.server_seq = 1;
                WAIT_TIME.store(1, Ordering::SeqCst);
            } else if data.status == HandshakeStatus::Logined as i32 {
                bail!(SocketError::AlreadyLogin);
            } else {
                unreachable!();
            }
        }
    }
    if state.trace_id == HyUuid::nil() {
        bail!(SocketError::InvalidMessage);
    }

    let span = info_span!("Agent connection", trace_id = %state.trace_id);
    async move {
        let (shell_tx, mut shell_rx) = mpsc::unbounded_channel();
        state.output = Some(shell_tx);
        if let Err(e) = frame
            .send_msg(&state.new_client_msg(Data::Info(InfoMessage {
                endpoint: addr.to_string(),
                version: env!("CARGO_PKG_VERSION").to_owned(),
                os: Some(consts::OS.to_owned()),
                system: System::long_os_version(),
                arch: Some(consts::ARCH.to_owned()),
                hostname: System::host_name(),
                report_rate: state.args.report_rate,
                disable_shell: state.args.disable_shell,
                ip: state.args.ip.map(|x| x.to_string()),
            })))
            .await
        {
            debug!(error = %e, "Error send info");
        }
        loop {
            select! {
                msg = shell_rx.recv() => {
                    if let Some(data) = msg {
                        frame.send_msg(&state.new_client_msg(data))
                        .await
                        .unwrap_or_else(|e| debug!(error = %e, "Send shell output error"));
                    }
                },
                msg = frame.read_msg() => {
                    let msg = msg?;
                    match handle_msg(&mut frame, &mut state, msg).await {
                        Ok(ret) => {
                            if let Some(ret) = ret {
                                return ret;
                            }
                        }
                        Err(e) =>  debug!(error = %e, "Error handle message"),
                    }
                }
            }
        }
    }
    .instrument(span)
    .await
}

pub async fn run(args: RunArgs, pubkey: PublicKey) {
    if args.passive {
        let listener = TcpListener::bind(&args.addr).await.unwrap();
        info!("Agent listening on {}", args.addr);
        let mut connected = false;
        let (tx, mut rx) = unbounded_channel::<bool>();
        loop {
            select! {
                c = listener.accept() => {
                     match c {
                        Ok((stream, addr)) => {
                            if !connected {
                                connected = true;
                                let tx = tx.clone();
                                let state = ConnectionState::new(args.clone());
                                spawn(async move {
                                    info!("Connection received {addr}");
                                    if let Err(e) = connect(stream, addr, &pubkey, state).await {
                                        error!("{e}");
                                        warn!("Agent is running in passive mode, reconnection depends on the server.");
                                    } else {
                                        let _ = tx.send(true);
                                    }
                                    let _ = tx.send(false);
                                });
                            }
                        }
                        Err(e) => error!("{e}"),
                    }
                }
                exit = rx.recv() => {
                    if exit.is_some_and(|x| x) {
                        break;
                    }else{
                        connected = false;
                    }
                }
            }
        }
    } else {
        loop {
            let state = ConnectionState::new(args.clone());
            if let Err(e) = active(&args.addr, &pubkey, state).await {
                if e.to_string() == SocketError::Reconnect.to_string() {
                    warn!("{e}");
                } else {
                    error!("{e}");
                }
                let wt = WAIT_TIME.load(Ordering::SeqCst);
                info!("Wait for {} seconds to reconnect", wt);
                sleep(Duration::from_secs(wt.into())).await;
                WAIT_TIME.store(min(args.max_time, wt * 2), Ordering::SeqCst);
            } else {
                break;
            }
        }
    }
}
