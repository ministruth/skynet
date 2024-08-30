use serde::Serializer;

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
pub fn vec_string_option<S>(data: &Option<Vec<u8>>, serializer: S) -> Result<S::Ok, S::Error>
where
    S: Serializer,
{
    match data {
        Some(x) => vec_string(x, serializer),
        None => serializer.serialize_none(),
    }
}
