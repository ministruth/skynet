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
    pub const fn new_rsp(id: &HyUuid, data: DataType) -> Self {
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
    Info(Info),
    Status(Status),
    ShellConnect(ShellConnect),
    ShellOutput(ShellOutput),
}

#[derive(Serialize, Deserialize, Debug)]
pub struct ShellConnect {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub token: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct ShellOutput {
    #[serde(skip_serializing_if = "String::is_empty")]
    pub token: String,
    pub data: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Login {
    pub uid: String,
    pub token: String,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Info {
    pub version: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub os: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub system: Option<String>,
    pub arch: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hostname: Option<String>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct Status {
    pub time: i64,
    pub cpu: f32,
    pub memory: u64,
    pub total_memory: u64,
    pub disk: u64,
    pub total_disk: u64,
    pub band_up: u64,
    pub band_down: u64,
}
