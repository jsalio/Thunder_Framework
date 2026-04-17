use std::path::Path;
use std::sync::mpsc::Sender;
use std::time::{Duration, Instant};

use notify::{Config, Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher};

use crate::event::{FileEvent, WatcherEvent};

// Debounce windows.
// Go files: 500ms — editors (VS Code Go extension) write the file twice on save
//   (raw content first, then gofmt-formatted version). Without debouncing we'd
//   trigger two back-to-back builds from a single save.
// Asset files: 100ms — no server restart involved, safe to fire quickly.
const DEBOUNCE_GO_MS: u64 = 500;
const DEBOUNCE_ASSET_MS: u64 = 100;

/// Classifies a changed path into a FileEvent based on its extension.
/// Returns None for paths we don't care about (e.g. editor swap files, target/).
pub fn classify(path: &Path, extensions: &[String]) -> Option<FileEvent> {
    let ext = path.extension()?.to_str()?;

    // Only react to extensions we're configured to watch.
    if !extensions.iter().any(|e| e == ext) {
        return None;
    }

    if ext == "go" {
        Some(FileEvent::GoChanged)
    } else {
        Some(FileEvent::AssetChanged)
    }
}

/// Starts the file watcher on a background thread.
/// Sends `WatcherEvent::FileChanged` into `tx` after debouncing.
pub fn start(watch_dir: &Path, extensions: Vec<String>, tx: Sender<WatcherEvent>) {
    let watch_dir = watch_dir.to_path_buf();

    std::thread::spawn(move || {
        // notify v6: create watcher with a std::sync::mpsc channel internally.
        // RecommendedWatcher resolves to FSEvents on macOS, inotify on Linux.
        let (notify_tx, notify_rx) = std::sync::mpsc::channel::<notify::Result<Event>>();

        let mut watcher = RecommendedWatcher::new(notify_tx, Config::default())
            .expect("failed to create file watcher");

        watcher
            .watch(&watch_dir, RecursiveMode::Recursive)
            .expect("failed to watch directory");

        // Pending debounce deadlines — one per event class.
        let mut pending_go: Option<Instant> = None;
        let mut pending_asset: Option<Instant> = None;

        loop {
            // Block until an event arrives or a debounce deadline expires.
            let timeout = next_deadline(pending_go, pending_asset);
            let result = notify_rx.recv_timeout(timeout);

            match result {
                Ok(Ok(event)) => {
                    // Only react to actual file modifications/creations.
                    if !is_relevant_kind(&event.kind) {
                        continue;
                    }
                    for path in &event.paths {
                        match classify(path, &extensions) {
                            Some(FileEvent::GoChanged) => {
                                pending_go = Some(Instant::now() + Duration::from_millis(DEBOUNCE_GO_MS));
                            }
                            Some(FileEvent::AssetChanged) => {
                                pending_asset = Some(Instant::now() + Duration::from_millis(DEBOUNCE_ASSET_MS));
                            }
                            None => {}
                        }
                    }
                }
                Ok(Err(e)) => {
                    eprintln!("[thunder] watcher error: {e}");
                }
                Err(_timeout) => {
                    // recv_timeout returned — check if any deadlines have passed.
                }
            }

            let now = Instant::now();

            if pending_go.is_some_and(|deadline| now >= deadline) {
                pending_go = None;
                let _ = tx.send(WatcherEvent::FileChanged(FileEvent::GoChanged));
            }

            if pending_asset.is_some_and(|deadline| now >= deadline) {
                pending_asset = None;
                let _ = tx.send(WatcherEvent::FileChanged(FileEvent::AssetChanged));
            }
        }
    });
}

/// Returns how long to block on recv_timeout before we need to check deadlines.
fn next_deadline(go: Option<Instant>, asset: Option<Instant>) -> Duration {
    let now = Instant::now();
    let candidates = [go, asset].into_iter().flatten();

    candidates
        .filter_map(|deadline| deadline.checked_duration_since(now))
        .min()
        .unwrap_or(Duration::from_secs(1))
}

/// Only react to modify/create events — ignore access, metadata, remove.
fn is_relevant_kind(kind: &EventKind) -> bool {
    matches!(
        kind,
        EventKind::Modify(_) | EventKind::Create(_)
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;

    fn exts() -> Vec<String> {
        vec!["go".into(), "html".into(), "css".into(), "js".into()]
    }

    #[test]
    fn go_file_is_go_changed() {
        let path = PathBuf::from("components/home.go");
        assert!(matches!(classify(&path, &exts()), Some(FileEvent::GoChanged)));
    }

    #[test]
    fn html_file_is_asset_changed() {
        let path = PathBuf::from("components/home.html");
        assert!(matches!(classify(&path, &exts()), Some(FileEvent::AssetChanged)));
    }

    #[test]
    fn css_file_is_asset_changed() {
        let path = PathBuf::from("components/home.css");
        assert!(matches!(classify(&path, &exts()), Some(FileEvent::AssetChanged)));
    }

    #[test]
    fn untracked_extension_is_none() {
        let path = PathBuf::from("notes.md");
        assert!(classify(&path, &exts()).is_none());
    }

    #[test]
    fn no_extension_is_none() {
        let path = PathBuf::from("Makefile");
        assert!(classify(&path, &exts()).is_none());
    }

    #[test]
    fn extra_ext_is_tracked() {
        let mut exts = exts();
        exts.push("toml".into());
        let path = PathBuf::from("config.toml");
        assert!(matches!(classify(&path, &exts), Some(FileEvent::AssetChanged)));
    }
}
