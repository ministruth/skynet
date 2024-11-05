use actix_cloud::response::{JsonResponse, RspResult};
use serde::Serialize;
use skynet_api::request::Request;

use crate::finish_data;

pub async fn get_public(req: Request) -> RspResult<JsonResponse> {
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
        recaptcha_enable: req.skynet.config.recaptcha.enable,
        recaptcha_url: req.skynet.config.recaptcha.url.clone(),
        recaptcha_sitekey: req.skynet.config.recaptcha.sitekey.clone(),
        lang: req.skynet.config.lang.clone(),
    };
    finish_data!(ret);
}
