use std::{any::Any, collections::HashMap, sync::Arc};

use derivative::Derivative;
use semver::{Version, VersionReq};

use crate::HyUuid;

struct APIItem {
    version: Version,
    item: Arc<Box<dyn Any + Send + Sync>>,
}

#[derive(Derivative)]
#[derivative(Default(new = "true"))]
pub struct APIManager {
    api: HashMap<HyUuid, APIItem>,
}

impl APIManager {
    /// Get plugin api by version.
    ///
    /// `ver` follows typical semver comparators, we strongly suggest only accept these two types:
    /// - `1.0.0`: exact match
    /// - `~1.0.0`: >=1.0.0 and <1.1.0
    ///
    /// # Panics
    ///
    /// Panics if `ver` is invalid.
    pub fn get<T: Any>(&self, id: &HyUuid, ver: &str) -> Option<Arc<Box<T>>> {
        let ver = VersionReq::parse(ver).unwrap();
        self.api.get(id).and_then(|x| {
            if ver.matches(&x.version) {
                let item = unsafe {
                    let raw = Arc::into_raw(x.item.clone()) as *const Box<T>;
                    Arc::from_raw(raw)
                };
                Some(item)
            } else {
                None
            }
        })
    }

    pub fn find(&self, id: &HyUuid, ver: &str) -> bool {
        let ver = VersionReq::parse(ver).unwrap();
        self.api.get(id).is_some_and(|x| ver.matches(&x.version))
    }

    /// Set plugin api.
    ///
    /// # Panics
    ///
    /// Panics if `ver` is invalid.
    pub fn set(&mut self, id: &HyUuid, ver: &str, item: Box<dyn Any + Send + Sync>) {
        self.api.insert(
            *id,
            APIItem {
                version: Version::parse(ver).unwrap(),
                item: Arc::new(item),
            },
        );
    }

    /// Clear all apis.
    pub fn clear(&mut self) {
        self.api.clear();
    }
}
