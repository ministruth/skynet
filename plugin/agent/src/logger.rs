use std::path::PathBuf;

use skynet_api::{
    actix_cloud::logger::{LogItem, Logger, LoggerBuilder, LoggerGuard},
    tracing::Level,
};

fn filter(item: &LogItem) -> bool {
    if item.target.starts_with("tungstenite::") {
        return false;
    }
    true
}

fn transformer(mut item: LogItem) -> LogItem {
    if item.target.starts_with("agent::") {
        item.target = String::from("agent");
    }

    // Remove useless file path.
    // Only keep path starting from the last `src`.
    if let Some(filename) = &item.filename {
        let mut buf = Vec::new();
        let mut flag = false;
        let s = PathBuf::from(filename);
        for i in s.iter().rev() {
            buf.push(i);
            if flag {
                break;
            }
            if i.eq_ignore_ascii_case("src") {
                flag = true;
            }
        }
        let mut s = PathBuf::new();
        for i in buf.into_iter().rev() {
            s.push(i);
        }
        item.filename = Some(s.to_string_lossy().into());
    }

    item
}

pub fn start_logger(
    enable: bool,
    json: bool,
    verbose: bool,
) -> (Option<Logger>, Option<LoggerGuard>) {
    if enable {
        let mut builder = LoggerBuilder::new();
        if json {
            builder = builder.json();
        }
        if verbose {
            builder = builder.filename().line_number().level(Level::DEBUG);
        }
        builder = builder.transformer(transformer).filter(filter);
        let (logger, guard) = builder.start();
        (Some(logger), Some(guard))
    } else {
        (None, None)
    }
}
