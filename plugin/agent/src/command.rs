use skynet_api::{actix_cloud::tokio::sync::mpsc::UnboundedSender, parking_lot::Mutex, Result};
use skynet_api_monitor::{message::Data, CommandRspMessage};
use std::{
    io::{ErrorKind, Read},
    sync::Arc,
    thread,
};
use subprocess::{Exec, ExitStatus, Popen, Redirection};

pub struct CommandInstance {
    process: Arc<Mutex<Popen>>,
}

impl CommandInstance {
    pub fn new(id: &str, cmd: &str, sender: Option<UnboundedSender<Data>>) -> Result<Self> {
        let mut process = Exec::shell(cmd)
            .stdout(Redirection::Pipe)
            .stderr(Redirection::Merge)
            .popen()?;
        let mut reader = process.stdout.take().unwrap();
        let process = Arc::new(Mutex::new(process));
        let ret = Self {
            process: process.clone(),
        };
        let id = id.to_owned();
        thread::spawn(move || {
            loop {
                let mut buffer = [0; 64];
                match reader.read(&mut buffer) {
                    Ok(n) => {
                        if n == 0 {
                            break;
                        }
                        if let Some(x) = &sender {
                            let _ = x.send(Data::CommandRsp(CommandRspMessage {
                                id: id.clone(),
                                code: None,
                                output: buffer[..n].to_vec(),
                            }));
                        }
                    }
                    Err(e) => {
                        if e.kind() != ErrorKind::Interrupted {
                            break;
                        }
                    }
                }
            }
            let code = process
                .lock()
                .wait()
                .map(|c| match c {
                    ExitStatus::Exited(c) => c.try_into().unwrap_or(-1),
                    ExitStatus::Signaled(c) => c.into(),
                    _ => -1,
                })
                .unwrap_or(-1);
            if let Some(x) = &sender {
                let _ = x.send(Data::CommandRsp(CommandRspMessage {
                    id,
                    code: Some(code),
                    output: Vec::new(),
                }));
            }
        });
        Ok(ret)
    }

    pub fn kill(&self, force: bool) -> Result<()> {
        if force {
            self.process.lock().kill().map_err(Into::into)
        } else {
            self.process.lock().terminate().map_err(Into::into)
        }
    }
}
