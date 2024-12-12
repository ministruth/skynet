use actix_cloud::{
    actix_web::web::Data,
    response::{JsonResponse, RspResult},
};
use serde::Serialize;
use skynet_api::Skynet;

use crate::finish_data;

pub async fn get_public(skynet: Data<Skynet>) -> RspResult<JsonResponse> {
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
        #[serde(rename(serialize = "geoip.enable"))]
        geoip_enable: bool,
    }
    let ret = Rsp {
        recaptcha_enable: skynet.config.recaptcha.enable,
        recaptcha_url: skynet.config.recaptcha.url.clone(),
        recaptcha_sitekey: skynet.config.recaptcha.sitekey.clone(),
        lang: skynet.config.lang.clone(),
        geoip_enable: skynet.config.geoip.enable,
    };
    finish_data!(ret);
}
