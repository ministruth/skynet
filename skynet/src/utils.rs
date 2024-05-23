use std::ffi::OsString;
use std::fs::{self, File};
use std::hash::Hash;
use std::io;
use std::path::PathBuf;
use std::{collections::HashSet, io::Cursor};

use anyhow::{bail, Result};
use base64::engine::general_purpose::STANDARD;
use base64::Engine;
use rand::{
    distributions::{Alphanumeric, Uniform},
    thread_rng, Rng,
};
use serde::Serializer;

pub fn is_unique<T>(iter: T) -> bool
where
    T: IntoIterator,
    T::Item: Eq + Hash,
{
    let mut uniq = HashSet::new();
    iter.into_iter().all(move |x| uniq.insert(x))
}

pub fn is_default<T: Default + PartialEq>(t: &T) -> bool {
    *t == Default::default()
}

/// Get `n` bytes random string.
/// `[a-zA-Z0-9]+`
#[must_use]
pub fn rand_string(n: usize) -> String {
    thread_rng()
        .sample_iter(&Alphanumeric)
        .take(n)
        .map(char::from)
        .collect()
}

/// Get `n` bytes random string (all printable ascii).
#[must_use]
pub fn rand_string_all(n: usize) -> String {
    thread_rng()
        .sample_iter(Uniform::new(char::from(33), char::from(126)))
        .take(n)
        .map(char::from)
        .collect()
}

#[must_use]
pub fn like_escape(s: &str) -> String {
    let mut s = s.replace('\\', "\\\\"); // first replace \
    s = s.replace('%', "\\%");
    s.replace('_', "\\_")
}

/// # Errors
/// No error will be returned.
pub fn vec_string<S>(data: &[u8], serializer: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    serializer.serialize_str(&String::from_utf8_lossy(data))
}

/// # Errors
/// No error will be returned.
pub fn osstring_lossy<S>(data: &OsString, serializer: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    serializer.serialize_str(&data.to_string_lossy())
}

/// # Errors
/// No error will be returned.
pub fn vec_string_option<S>(data: &Option<Vec<u8>>, serializer: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    match data {
        Some(x) => vec_string(x, serializer),
        None => serializer.serialize_none(),
    }
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
        let mut archive = zip::ZipArchive::new(Cursor::new(data))?;
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
    func().map_err(|e| {
        let _ = fs::remove_dir_all(path);
        e
    })
}
