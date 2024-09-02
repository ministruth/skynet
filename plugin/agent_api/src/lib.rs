use enum_as_inner::EnumAsInner;
use skynet_api::{uuid, HyUuid};
use std::{
    fmt::{self, Display},
    fs,
    path::PathBuf,
};
use version_compare::{compare, Cmp};

pub const VERSION: &str = env!("CARGO_PKG_VERSION");
pub const ID: HyUuid = HyUuid(uuid!("ce96ae04-6801-4ca4-b09d-a087e05f3783"));

#[derive(EnumAsInner, Clone, Copy, Debug)]
#[repr(u8)]
pub enum System {
    Windows,
    Linux,
    OSX,
}

impl System {
    #[must_use]
    pub fn parse(str: &str) -> Option<Self> {
        let str = str.to_lowercase();
        if str.contains("windows") {
            Some(Self::Windows)
        } else if str.contains("linux") {
            Some(Self::Linux)
        } else if str.contains("macos") {
            Some(Self::OSX)
        } else {
            None
        }
    }
}

impl Display for System {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Windows => write!(f, "windows"),
            Self::Linux => write!(f, "linux"),
            Self::OSX => write!(f, "osx"),
        }
    }
}

#[derive(EnumAsInner, Clone, Copy, Debug)]
#[repr(u8)]
pub enum Arch {
    X86,
    X64,
    ARM,
    ARM64,
}

impl Arch {
    #[must_use]
    pub fn parse(str: &str) -> Option<Self> {
        let str = str.to_lowercase();
        if str.contains("x86_64") {
            Some(Self::X64)
        } else if str.contains("x86") {
            Some(Self::X86)
        } else if str.contains("aarch64") {
            Some(Self::ARM64)
        } else if str.contains("arm") {
            Some(Self::ARM)
        } else {
            None
        }
    }
}

impl Display for Arch {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::X86 => write!(f, "x86"),
            Self::X64 => write!(f, "x64"),
            Self::ARM => write!(f, "arm"),
            Self::ARM64 => write!(f, "arm64"),
        }
    }
}

#[derive(Debug)]
pub struct Service {
    path: PathBuf,
    version: String,
}

impl Service {
    #[must_use]
    pub const fn new(path: PathBuf, version: String) -> Self {
        Self { path, version }
    }

    #[must_use]
    pub fn check_version(&self, v: &str) -> bool {
        compare(&self.version, v) == Ok(Cmp::Gt)
    }

    #[must_use]
    pub fn get_binary_name(&self, sys: System, arch: Arch) -> PathBuf {
        let suffix = if sys.is_windows() { ".exe" } else { "" };
        self.path
            .join("bin")
            .join(format!("agent_{sys}_{arch}{suffix}"))
    }

    #[must_use]
    pub fn get_binary(&self, sys: System, arch: Arch) -> Option<Vec<u8>> {
        let suffix = if sys.is_windows() { ".exe" } else { "" };
        fs::read(
            self.path
                .join("bin")
                .join(format!("agent_{sys}_{arch}{suffix}")),
        )
        .ok()
    }
}
