use actix_web::{web::Data, Responder};
use serde_json::{Map, Number, Value};
use skynet::{
    config::ConfigItem,
    finish,
    request::{Response, RspResult},
    Skynet,
};

pub async fn get_public(skynet: Data<Skynet>) -> RspResult<impl Responder> {
    let mut ret: Map<String, Value> = Map::new();
    for (_, v) in skynet.config.iter() {
        if let Some(v) = v.downcast_ref::<ConfigItem<String>>() {
            if v.public {
                ret.insert(v.name.clone(), Value::String(v.get().to_owned()));
            }
        } else if let Some(v) = v.downcast_ref::<ConfigItem<i64>>() {
            if v.public {
                ret.insert(v.name.clone(), Value::Number(v.get().into()));
            }
        } else if let Some(v) = v.downcast_ref::<ConfigItem<bool>>() {
            if v.public {
                ret.insert(v.name.clone(), Value::Bool(v.get()));
            }
        } else if let Some(v) = v.downcast_ref::<ConfigItem<f64>>() {
            if v.public {
                ret.insert(
                    v.name.clone(),
                    Value::Number(Number::from_f64(v.get()).unwrap()),
                );
            }
        } else {
            unreachable!()
        }
    }
    finish!(Response::data(ret));
}
