use std::collections::hash_map::DefaultHasher;
use std::hash::{Hash, Hasher};
use std::io::{BufRead, BufReader};
use std::path::{Path, PathBuf};
use std::process::{Child, Command, Stdio};

#[cfg(unix)]
use libc;
use std::sync::mpsc::SyncSender;
use std::time::{Duration, Instant};

use tokio::sync::mpsc::Sender;

use crate::event::WatcherEvent;
use crate::platform::Platform;

// ── Build ────────────────────────────────────────────────────────────────────

/// Runs `go build -o <output> <package>`.
/// Returns `Err(stderr)` on non-zero exit — the caller decides what to do with
/// the running server; this function never touches a child process.
pub fn build(package: &str, output: &Path) -> Result<(), String> {
    let result = Command::new("go")
        .args(["build", "-o", output.to_str().unwrap(), package])
        .output()
        .map_err(|e| format!("failed to run go build: {e}"))?;

    if result.status.success() {
        Ok(())
    } else {
        Err(String::from_utf8_lossy(&result.stderr).into_owned())
    }
}

/// Returns the path where the temp binary for this package should be written.
/// Format: `$TMPDIR/thunder-<hash>`, with the platform extension applied.
pub fn temp_binary_path(package: &str, platform: &dyn Platform) -> PathBuf {
    let mut hasher = DefaultHasher::new();
    package.hash(&mut hasher);
    let hash = hasher.finish();

    let base = std::env::temp_dir().join(format!("thunder-{hash:016x}"));
    platform.binary_path(&base)
}

// ── Kill sequence ────────────────────────────────────────────────────────────

/// Gracefully stops a process by PID.
/// Sequence: SIGTERM → poll liveness up to 3s → SIGKILL.
/// Zombie reaping is handled separately by the monitor task that holds the Child.
pub fn kill_by_pid(pid: u32, platform: &dyn Platform) {
    let _ = platform.terminate_pid(pid);

    let deadline = Instant::now() + Duration::from_secs(3);
    while Instant::now() < deadline {
        if !is_pid_alive(pid) {
            return; // exited cleanly after SIGTERM
        }
        std::thread::sleep(Duration::from_millis(50));
    }

    let _ = platform.force_kill_pid(pid);
}

/// Returns false once the process is no longer alive.
/// On Unix: signal 0 probes existence without sending an actual signal.
/// On other platforms: conservatively returns true (force_kill_pid will fire).
#[cfg(unix)]
fn is_pid_alive(pid: u32) -> bool {
    // SAFETY: signal 0 never delivers a signal; it only checks if the PID exists.
    unsafe { libc::kill(pid as i32, 0) == 0 }
}

#[cfg(not(unix))]
fn is_pid_alive(_pid: u32) -> bool {
    true
}

// ── Spawn ────────────────────────────────────────────────────────────────────

/// Spawns the compiled binary as a child process.
/// Sets `THUNDER_WATCHER=1` and `THUNDER_WS_PORT=<ws_port>` so the Go side
/// knows to inject the live-reload script.
/// Pipes stdout so `read_stdout` can extract the port and forward logs.
pub fn spawn(binary: &Path, ws_port: u16) -> std::io::Result<Child> {
    Command::new(binary)
        .env("THUNDER_WATCHER", "1")
        .env("THUNDER_WS_PORT", ws_port.to_string())
        .stdout(Stdio::piped())
        .stderr(Stdio::inherit())
        .spawn()
}

// ── Stdout reader ────────────────────────────────────────────────────────────

/// Reads child stdout line by line on a plain (non-async) thread.
/// - Forwards every line to the terminal.
/// - On match for `addr=<port>` in Thunder's startup log: sends the port back
///   via a oneshot channel so the reactor can start port polling.
///
/// This blocks forever (until the child closes stdout), so it MUST run in a
/// dedicated `std::thread::spawn`, never in an async context.
pub fn read_stdout(
    stdout: std::process::ChildStdout,
    port_tx: SyncSender<u16>,
) {
    std::thread::spawn(move || {
        let reader = BufReader::new(stdout);
        let mut port_sent = false;

        for line in reader.lines() {
            let Ok(line) = line else { break };
            println!("{line}");

            if !port_sent {
                if let Some(port) = extract_port(&line) {
                    let _ = port_tx.send(port);
                    port_sent = true;
                }
            }
        }
    });
}

/// Parses Thunder's startup log line to extract the port.
/// Matches: `msg="server starting" addr=<port>` (slog format)
/// Also handles plain `addr=:<port>` with a leading colon.
fn extract_port(line: &str) -> Option<u16> {
    // Look for `addr=` followed by an optional `:` and digits.
    let idx = line.find("addr=")?;
    let rest = &line[idx + 5..]; // skip "addr="
    let rest = rest.trim_start_matches(':'); // strip optional leading colon
    let port_str = rest.split_whitespace().next()?.trim_matches('"');
    port_str.parse().ok()
}

// ── Port polling ─────────────────────────────────────────────────────────────

/// Polls `127.0.0.1:<port>` every 50ms until the server accepts connections.
/// Sends `WatcherEvent::PortReady` on success, `WatcherEvent::PortTimeout` after 10s.
/// Runs as a `tokio::spawn` task — result feeds back into the unified reactor channel.
pub async fn poll_port(port: u16, tx: Sender<WatcherEvent>) {
    let addr = format!("127.0.0.1:{port}");
    let deadline = tokio::time::Instant::now() + Duration::from_secs(10);

    loop {
        if tokio::time::Instant::now() >= deadline {
            let _ = tx.send(WatcherEvent::PortTimeout).await;
            return;
        }

        match tokio::net::TcpStream::connect(&addr).await {
            Ok(_) => {
                let _ = tx.send(WatcherEvent::PortReady).await;
                return;
            }
            Err(_) => {
                tokio::time::sleep(Duration::from_millis(50)).await;
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn extract_port_slog_format() {
        let line = r#"time=2024-01-01T00:00:00Z level=INFO msg="server starting" addr=8090"#;
        assert_eq!(extract_port(line), Some(8090));
    }

    #[test]
    fn extract_port_with_colon() {
        let line = r#"msg="server starting" addr=:8086"#;
        assert_eq!(extract_port(line), Some(8086));
    }

    #[test]
    fn extract_port_no_match() {
        let line = "something else entirely";
        assert_eq!(extract_port(line), None);
    }
}
