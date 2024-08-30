use std::{io::Result, path::PathBuf};
use walkdir::WalkDir;

fn main() -> Result<()> {
    prost_build::compile_protos(
        &WalkDir::new("proto")
            .into_iter()
            .filter(|e| {
                e.as_ref()
                    .is_ok_and(|e| e.path().extension().is_some_and(|e| e == "proto"))
            })
            .map(|e| e.unwrap().into_path())
            .collect::<Vec<PathBuf>>(),
        &["proto"],
    )?;
    Ok(())
}
