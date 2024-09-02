use std::{
    io::{ErrorKind, Read, Write},
    thread,
};

use portable_pty::{native_pty_system, Child, CommandBuilder, PtyPair, PtySize};
use skynet_api::actix_cloud::tokio::sync::mpsc::UnboundedSender;
use skynet_api::Result;
use skynet_api_monitor::{message::Data, ShellOutputMessage};

pub struct ShellInstance {
    writer: Box<dyn Write + Send>,
    pair: PtyPair,
    child: Box<dyn Child + Send + Sync>,
}

impl ShellInstance {
    pub fn new(
        token: &str,
        cmd: &str,
        rows: u16,
        cols: u16,
        sender: Option<UnboundedSender<Data>>,
    ) -> Result<Self> {
        let pty_system = native_pty_system();
        let pair = pty_system.openpty(PtySize {
            pixel_width: 0,
            pixel_height: 0,
            rows,
            cols,
        })?;
        let child = pair.slave.spawn_command(CommandBuilder::new(cmd))?;
        let mut reader = pair.master.try_clone_reader()?;
        let writer = pair.master.take_writer()?;

        // safe to detach, terminated when reader closed.
        let token = token.to_owned();
        thread::spawn(move || loop {
            let mut buffer = [0; 64];
            match reader.read(&mut buffer) {
                Ok(n) => {
                    if n == 0 {
                        break;
                    }
                    if let Some(x) = &sender {
                        let _ = x.send(Data::ShellOutput(ShellOutputMessage {
                            token: Some(token.clone()),
                            data: buffer[..n].to_vec(),
                        }));
                    }
                }
                Err(e) => {
                    if e.kind() != ErrorKind::Interrupted {
                        break;
                    }
                }
            }
        });
        Ok(Self {
            writer,
            pair,
            child,
        })
    }

    pub fn resize(&self, rows: u16, cols: u16) -> Result<()> {
        self.pair.master.resize(PtySize {
            pixel_width: 0,
            pixel_height: 0,
            rows,
            cols,
        })
    }

    pub fn kill(&mut self) -> Result<()> {
        self.child.kill().map_err(Into::into)
    }

    pub fn write(&mut self, data: &[u8]) -> Result<()> {
        Ok(self.writer.write_all(data)?)
    }
}

impl Drop for ShellInstance {
    fn drop(&mut self) {
        let _ = self.kill();
    }
}
