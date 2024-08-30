use serde::Serialize;
use skynet_api::{
    actix_cloud::{
        actix_web::{web::Data, Responder},
        response::RspResult,
    },
    Skynet,
};

use crate::finish_data;

pub async fn get_public(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    #[derive(Serialize)]
    struct Rsp {
        #[serde(rename(serialize = "recaptcha.enable"))]
        recaptcha_enable: bool,
        #[serde(rename(serialize = "recaptcha.url"))]
        recaptcha_url: String,
        #[serde(
            rename(serialize = "recaptcha.sitekey"),
            skip_serializing_if = "Option::is_none"
        )]
        recaptcha_sitekey: Option<String>,
        lang: String,
    }
    let ret = Rsp {
        recaptcha_enable: skynet.config.recaptcha.enable,
        recaptcha_url: skynet.config.recaptcha.url.clone(),
        recaptcha_sitekey: skynet.config.recaptcha.sitekey.clone(),
        lang: skynet.config.lang.clone(),
    };
    finish_data!(ret);
}
