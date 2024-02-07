use std::{
    cmp::min,
    env::{self, consts},
    fs,
    os::unix::process::CommandExt,
    process::Command,
    time::Duration,
};

use anyhow::{bail, Result};
use base64::{engine::general_purpose::STANDARD, Engine};
use futures_util::{SinkExt, StreamExt};
use log::{debug, error, info, warn};
use miniz_oxide::inflate::decompress_to_vec;
use monitor_service::{client, server};
use sysinfo::System;
use tokio::time::sleep;
use tokio_tungstenite::{
    connect_async_with_config,
    tungstenite::{protocol::WebSocketConfig, Message},
};
use url::Url;

use crate::get_uid;

async fn connect(addr: &str, token: &str, wait_time: &mut u32) -> Result<()> {
    let url = Url::parse(&format!("{addr}/api/plugins/{}/ws", monitor_service::ID)).unwrap();
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

    let mut is_login = false;
    let login_msg = server::Message::new(server::DataType::Login(server::Login {
        uid: get_uid(),
        token: token.to_owned(),
    }));
    ws.send(Message::Text(login_msg.json())).await?;
    while let Some(msg) = ws.next().await {
        match msg {
            Ok(msg) => match msg {
                Message::Text(x) => match serde_json::from_str::<client::Message>(&x) {
                    Ok(msg) => match msg.data {
                        client::DataType::Login(data) => {
                            if is_login {
                                if data.code != 0 {
                                    warn!("Msg `{}` code `{}`: {}", msg.id, data.code, data.msg);
                                }
                            } else if msg.id != login_msg.id {
                                bail!("Invalid login response");
                            } else if data.code != 0 {
                                bail!("Login error: code `{}`: {}", data.code, data.msg);
                            } else {
                                is_login = true;
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
                            }
                        }
                        client::DataType::Update(data) => {
                            let exe = env::current_exe()?;
                            let file = STANDARD.decode(data.data)?;
                            let file = decompress_to_vec(&file)
                                .map_err(|e| anyhow::anyhow!(e.to_string()))?;
                            let crc = crc32fast::hash(&file);
                            if crc == data.crc32 {
                                let new_path = format!("_agent_update{}", consts::EXE_SUFFIX);
                                fs::write(&new_path, file)?;
                                self_replace::self_replace(&new_path)?;
                                fs::remove_file(new_path)?;
                                let _ = ws.close(None).await;
                                info!("Trigger update, crc32: {}", crc);
                                return Err(Command::new(exe)
                                    .args(env::args().skip(1))
                                    .exec()
                                    .into());
                            } else {
                                bail!("Update: CRC32 mismatch");
                            }
                        }
                        client::DataType::Quit => {
                            info!("Receive quit signal from server");
                            return Ok(());
                        }
                    },
                    Err(e) => debug!("Parse message error: {e}"),
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
    }
    bail!("Connection lost");
}

#[allow(clippy::while_let_loop)]
pub async fn run(addr: String, token: String, max_time: u32) {
    let mut wait_time = 1;
    loop {
        if let Err(e) = connect(&addr, &token, &mut wait_time).await {
            error!("Fatal: {e}");
            info!("Wait for {wait_time} seconds to reconnect");
            sleep(Duration::from_secs(wait_time.into())).await;
            wait_time = min(max_time, wait_time * 2);
        } else {
            break;
        }
    }
}
