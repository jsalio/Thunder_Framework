use std::path::PathBuf;
use clap::{Parser, Subcommand};

#[derive(Parser, Debug)]
#[command(name = "thunder", about = "Thunder Framework CLI")]
pub struct Cli {
    #[command(subcommand)]
    pub command: Command,
}

#[derive(Subcommand, Debug)]
pub enum Command {
    /// Watch a Go package, rebuild on changes, and live-reload the browser.
    Watch(WatchArgs),
}

#[derive(Parser, Debug)]
pub struct WatchArgs {
    /// Path to the Go package to build and run.
    #[arg(default_value = ".")]
    pub go_package: String,

    /// Directory to watch for file changes. Defaults to go_package.
    #[arg(short = 'd', long)]
    pub watch_dir: Option<PathBuf>,

    /// WebSocket port for live-reload signals.
    #[arg(short = 'w', long, default_value_t = 3001)]
    pub ws_port: u16,

    /// Build new binary before killing the old server (keeps old server alive during compile).
    #[arg(long, default_value_t = false)]
    pub build_first: bool,

    /// Extra file extensions to watch, comma-separated (e.g. "toml,json").
    #[arg(short = 'e', long)]
    pub extra_ext: Option<String>,
}

impl WatchArgs {
    /// Returns the directory to watch. Falls back to go_package if --watch-dir not set.
    pub fn resolved_watch_dir(&self) -> PathBuf {
        self.watch_dir
            .clone()
            .unwrap_or_else(|| PathBuf::from(&self.go_package))
    }

    /// Returns the full set of extensions to watch.
    pub fn extensions(&self) -> Vec<String> {
        let mut exts = vec![
            "go".to_string(),
            "html".to_string(),
            "css".to_string(),
            "js".to_string(),
        ];
        if let Some(extra) = &self.extra_ext {
            for ext in extra.split(',') {
                let trimmed = ext.trim().to_string();
                if !trimmed.is_empty() {
                    exts.push(trimmed);
                }
            }
        }
        exts
    }
}
