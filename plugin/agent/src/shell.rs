use std::io::{ErrorKind, Read, Write};

use anyhow::Result;
use portable_pty::{native_pty_system, Child, CommandBuilder, PtyPair, PtySize};
use tokio::sync::mpsc::UnboundedSender;

pub struct ShellInstance {
    pub cmd: String,
    writer: Box<dyn Write + Send>,
    pair: PtyPair,
    child: Box<dyn Child + Send + Sync>,
}

impl ShellInstance {
    pub fn new(
        cmd: &str,
        rows: u16,
        cols: u16,
        sender: Option<UnboundedSender<Vec<u8>>>,
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
        tokio::spawn(async move {
            loop {
                let mut buffer = [0; 64];
                match reader.read(&mut buffer) {
                    Ok(n) => {
                        if n == 0 {
                            break;
                        }
                        if let Some(x) = &sender {
                            let _ = x.send(buffer[..n].to_vec());
                        }
                    }
                    Err(e) => {
                        if e.kind() != ErrorKind::Interrupted {
                            break;
                        }
                    }
                }
            }
        });
        Ok(Self {
            cmd: cmd.to_owned(),
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
