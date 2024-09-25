use std::fs::{self, File};
use std::io;
use std::io::Cursor;
use std::path::PathBuf;

use actix_cloud::{bail, Result};
use base64::engine::general_purpose::STANDARD;
use base64::Engine;
use zip::ZipArchive;

/// Check whether given `s` is a hex string.
pub fn is_hex(s: &str) -> bool {
    s.chars().all(|c| c.is_ascii_hexdigit())
}

#[must_use]
pub fn like_escape(s: &str) -> String {
    let mut s = s.replace('\\', "\\\\"); // first replace \
    s = s.replace('%', "\\%");
    s.replace('_', "\\_")
}

#[must_use]
pub fn get_dataurl(data: &[u8]) -> (String, Option<infer::Type>) {
    let mime = infer::get(data);
    mime.map_or((String::new(), None), |mime| {
        let data = STANDARD.encode(data);
        (format!("data:{mime};base64,{data}"), Some(mime))
    })
}

#[must_use]
pub fn parse_dataurl(data: &str) -> (Vec<u8>, Option<infer::Type>) {
    let data: Vec<&str> = data.split(',').collect();
    if data.len() == 2 {
        STANDARD.decode(data[1]).map_or((Vec::new(), None), |data| {
            let mime = infer::get(&data);
            (data, mime)
        })
    } else {
        (Vec::new(), None)
    }
}

/// Unzip `data` to `path`.
///
/// `path` must be a non-existing path. It will be automatically removed when return error.
///
/// # Errors
/// Will return `Err` when extract failed, with `path` removed.
pub fn unzip(data: &[u8], path: &PathBuf) -> Result<()> {
    if path.exists() {
        bail!("Cannot extract to existed folder");
    }
    let func = || -> Result<()> {
        let mut archive = ZipArchive::new(Cursor::new(data))?;
        for i in 0..archive.len() {
            let mut file = archive.by_index(i)?;
            let outpath = path.join(match file.enclosed_name() {
                Some(path) => path.clone(),
                None => continue,
            });

            if file.is_dir() {
                fs::create_dir_all(outpath)?;
            } else {
                if let Some(p) = outpath.parent() {
                    if !p.exists() {
                        fs::create_dir_all(p)?;
                    }
                }
                let mut outfile = File::create(&outpath)?;
                io::copy(&mut file, &mut outfile)?;
            }
        }
        Ok(())
    };
    func().inspect_err(|_| {
        let _ = fs::remove_dir_all(path);
    })
}
