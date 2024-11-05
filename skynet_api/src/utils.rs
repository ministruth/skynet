#[must_use]
pub fn like_escape(s: &str) -> String {
    let mut s = s.replace('\\', "\\\\"); // first replace \
    s = s.replace('%', "\\%");
    s.replace('_', "\\_")
}

/// Check whether given `s` is a hex string.
pub fn is_hex(s: &str) -> bool {
    s.chars().all(|c| c.is_ascii_hexdigit())
}

#[cfg(feature = "extra-utils")]
#[must_use]
pub fn get_dataurl(data: &[u8]) -> (String, Option<infer::Type>) {
    use base64::prelude::*;

    let mime = infer::get(data);
    mime.map_or((String::new(), None), |mime| {
        let data = BASE64_STANDARD.encode(data);
        (format!("data:{mime};base64,{data}"), Some(mime))
    })
}

#[cfg(feature = "extra-utils")]
#[must_use]
pub fn parse_dataurl(data: &str) -> (Vec<u8>, Option<infer::Type>) {
    use base64::prelude::*;

    let data: Vec<&str> = data.split(',').collect();
    if data.len() == 2 {
        BASE64_STANDARD
            .decode(data[1])
            .map_or((Vec::new(), None), |data| {
                let mime = infer::get(&data);
                (data, mime)
            })
    } else {
        (Vec::new(), None)
    }
}
