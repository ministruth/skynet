use std::{cmp::min, process::exit, time::Duration};

use anyhow::{bail, Result};
use futures_util::{SinkExt, StreamExt};
use log::{debug, error, info, warn};
use monitor::msg::{client, server};
use sysinfo::System;
use tokio::time::sleep;
use tokio_tungstenite::{connect_async, tungstenite::Message};
use url::Url;

use crate::get_uid;

async fn connect(addr: &str, token: &str, wait_time: &mut u32) -> Result<()> {
    let url = Url::parse(&format!(
        "{addr}/api/plugins/2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa/ws"
    ))
    .unwrap();
    info!("Connecting to {url}");
    let (mut ws, _) = connect_async(url).await?;
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
                                        os: System::name(),
                                        system: System::long_os_version(),
                                        machine: System::cpu_arch(),
                                        hostname: System::host_name(),
                                    }))
                                    .into(),
                                ))
                                .await
                                .unwrap_or_else(|e| debug!("Send info error: {e}"));
                            }
                        }
                        client::DataType::Quit => {
                            info!("Receive quit signal from server");
                            exit(0);
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
