use std::fmt::{self, Display};

use anyhow::{bail, Result};
#[cfg(feature = "database")]
use sea_orm::{
    prelude::*,
    sea_query::{ArrayType, Nullable, ValueType, ValueTypeErr},
    ColIdx, TryFromU64, TryGetError, TryGetable,
};
#[cfg(feature = "serde")]
use serde::{de, Deserialize, Deserializer, Serialize, Serializer};
use uuid::Uuid;

#[derive(Clone, Copy, Hash, Debug, PartialEq, Eq, Default)]
pub struct HyUuid(pub Uuid);

impl Display for HyUuid {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.hyphenated().fmt(f)
    }
}

impl HyUuid {
    pub const fn nil() -> Self {
        Self(Uuid::nil())
    }

    pub const fn is_nil(&self) -> bool {
        self.0.is_nil()
    }

    pub fn new() -> Self {
        Self(Uuid::new_v4())
    }

    /// # Errors
    /// Will return `Err` when `str` is not uuid v4.
    pub fn parse(str: &str) -> Result<Self> {
        if str.len() != 36 {
            bail!("uuid error: uuid must be 36 bytes");
        }
        Uuid::parse_str(str).map_or_else(|e| Err(e.into()), |v| Ok(Self(v)))
    }
}

pub fn uuids2strings(u: &[HyUuid]) -> Vec<String> {
    u.iter().map(ToString::to_string).collect()
}

#[cfg(feature = "database")]
impl Nullable for HyUuid {
    fn null() -> Value {
        Value::String(None)
    }
}

#[cfg(feature = "database")]
impl TryFromU64 for HyUuid {
    fn try_from_u64(_: u64) -> Result<Self, DbErr> {
        Err(DbErr::ConvertFromU64(stringify!(HyUuid)))
    }
}

#[cfg(feature = "database")]
impl ValueType for HyUuid {
    fn try_from(v: Value) -> Result<Self, ValueTypeErr> {
        ValueType::try_from(v).and_then(|v: String| Self::parse(&v).map_err(|_e| ValueTypeErr {}))
    }

    fn type_name() -> String {
        stringify!(HyUuid).to_owned()
    }

    fn array_type() -> ArrayType {
        ArrayType::Char
    }

    fn column_type() -> ColumnType {
        ColumnType::Char(Some(36))
    }
}

#[cfg(feature = "database")]
impl From<HyUuid> for Value {
    fn from(value: HyUuid) -> Self {
        Self::String(Some(Box::new(value.0.hyphenated().to_string())))
    }
}

#[cfg(feature = "database")]
impl TryGetable for HyUuid {
    fn try_get_by<I: ColIdx>(res: &QueryResult, index: I) -> Result<Self, TryGetError> {
        TryGetable::try_get_by(res, index).and_then(|v: String| {
            Self::parse(&v).map_err(|e| TryGetError::DbErr(DbErr::Type(e.to_string())))
        })
    }
}

#[cfg(feature = "serde")]
impl Serialize for HyUuid {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        serializer.serialize_str(self.0.hyphenated().encode_lower(&mut Uuid::encode_buffer()))
    }
}
#[cfg(feature = "serde")]
impl<'de> Deserialize<'de> for HyUuid {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        struct UuidVisitor;

        impl<'vi> de::Visitor<'vi> for UuidVisitor {
            type Value = HyUuid;

            fn expecting(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
                write!(formatter, "a UUID v4 string with hyphens")
            }

            fn visit_str<E>(self, value: &str) -> Result<Self::Value, E>
            where
                E: de::Error,
            {
                HyUuid::parse(value).map_err(|e| de::Error::custom(e))
            }
        }

        deserializer.deserialize_str(UuidVisitor)
    }
}
