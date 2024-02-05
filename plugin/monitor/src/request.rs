use skynet::{request::ResponseCodeTrait, t, Skynet};

#[derive(Debug, Clone, Copy)]
#[repr(u32)]
pub enum ResponseCode {
    CodeAgentExist = 10000,
}

impl ResponseCodeTrait for ResponseCode {
    fn translate(&self, skynet: &Skynet, locale: &str) -> String {
        t!(
            skynet,
            match self {
                Self::CodeAgentExist => "2eb2e1a5-66b4-45f9-ad24-3c4f05c858aa.response.agent.exist",
            },
            locale
        )
    }

    fn code(&self) -> u32 {
        *self as u32
    }
}
