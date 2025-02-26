use std::time::Duration;

use actix_cloud::actix_web::web::Data;
use reqwest::{Client, Proxy};
use serde::{Deserialize, Serialize};
use skynet_api::{Result, Skynet, bail};

fn create_client(skynet: Data<Skynet>) -> Result<Client> {
    let mut builder = Client::builder();
    if let Some(x) = &skynet.config.client.proxy {
        let mut proxy = Proxy::all(x)?;
        if let Some(user) = &skynet.config.client.username {
            let pass = skynet.config.client.password.as_deref().unwrap_or_default();
            proxy = proxy.basic_auth(user, pass);
        }
        builder = builder.proxy(proxy);
    }
    if skynet.config.client.timeout != 0 {
        builder = builder.timeout(Duration::from_secs(skynet.config.client.timeout.into()));
    }
    builder.build().map_err(Into::into)
}

#[derive(Debug)]
pub(crate) struct RecaptchaOption {
    pub(crate) url: String,
    pub(crate) secret: String,
}

pub(crate) async fn verify_recaptcha(
    skynet: Data<Skynet>,
    response: String,
    ip: String,
    option: RecaptchaOption,
) -> Result<()> {
    #[derive(Deserialize, Serialize)]
    struct Response {
        success: bool,
        #[serde(default)]
        challenge_ts: String,
        #[serde(default)]
        hostname: String,
        #[serde(default, rename = "error-codes")]
        error_codes: Vec<String>,
    }
    let client = create_client(skynet)?;
    let req = client
        .post(option.url + "/recaptcha/api/siteverify")
        .form(&[
            ("secret", option.secret),
            ("remoteip", ip),
            ("response", response),
        ]);
    let rsp = req.send().await?.json::<Response>().await?;
    if !rsp.error_codes.is_empty() {
        bail!("Remote error codes: {:?}", rsp.error_codes)
    }
    if !rsp.success {
        bail!("Invalid challenge solution or remote IP")
    }
    Ok(())
}
