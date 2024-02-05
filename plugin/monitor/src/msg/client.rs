use bytestring::ByteString;
use serde::{Deserialize, Serialize};
use serde_json::json;
use skynet::HyUuid;

#[derive(Serialize, Deserialize, Debug)]
pub struct Message {
    pub id: HyUuid,
    pub data: DataType,
}

impl Message {
    #[must_use]
    pub fn new(data: DataType) -> Self {
        Self {
            id: HyUuid::new(),
            data,
        }
    }

    #[must_use]
    pub fn new_rsp(id: &HyUuid, data: DataType) -> Self {
        Self { id: *id, data }
    }

    #[must_use]
    pub fn json(&self) -> String {
        json!(self).to_string()
    }
}

impl From<Message> for String {
    fn from(value: Message) -> Self {
        value.json()
    }
}

impl From<Message> for ByteString {
    fn from(value: Message) -> Self {
        value.json().into()
    }
}

#[derive(Serialize, Deserialize, Debug)]
#[serde(tag = "type")]
pub enum DataType {
    Login(Login),
    Quit,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Login {
    pub code: i32,
    pub msg: String,
}
