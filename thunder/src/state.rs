use std::path::PathBuf;

use crate::event::{FileEvent, SideEffect, WatcherEvent};

/// All possible states of the managed Go process.
/// No `Child` handles are stored here — they live in the reactor's monitor tasks.
/// PIDs are stored instead; killing is done via Platform::terminate_pid/force_kill_pid.
pub enum ProcessState {
    /// No process running. Waiting for the first file change.
    Idle,

    /// `go build` is in progress.
    /// `running`: pid of the old server kept alive during build-first mode, None otherwise.
    Building { running: Option<u32> },

    /// Build succeeded. Old process is being killed; new binary is ready to spawn.
    Stopping { binary: PathBuf },

    /// New binary spawned. Waiting to learn its pid (ProcessSpawned) then its port (PortReady).
    WaitingForPort { pid: Option<u32>, binary: PathBuf },

    /// Server is up and serving. Normal steady state.
    Running { pid: u32, binary: PathBuf },

    /// Build failed or server exited unexpectedly.
    /// `running`: pid of old server still alive (build-first), if any.
    Failed { reason: String, running: Option<u32> },
}

/// Pure state transition function — no I/O, no async, no side effects.
pub fn transition(
    state: ProcessState,
    event: WatcherEvent,
    build_first: bool,
    package: &str,
    binary: &PathBuf,
) -> (ProcessState, Option<SideEffect>) {
    use FileEvent::*;
    use ProcessState::*;
    use SideEffect::*;
    use WatcherEvent::*;

    match (state, event) {
        // ── Idle ─────────────────────────────────────────────────────────────
        (Idle, FileChanged(GoChanged)) => (
            Building { running: None },
            Some(SpawnBuild {
                package: package.to_string(),
                output: binary.clone(),
            }),
        ),

        (Idle, FileChanged(AssetChanged)) => (Idle, None),

        // ── Building ─────────────────────────────────────────────────────────
        (Building { running }, BuildComplete(Ok(bin))) => (
            Stopping { binary: bin },
            Some(KillAndSpawn { pid: running }),
        ),

        (Building { running }, BuildComplete(Err(msg))) => (
            Failed { reason: msg.clone(), running },
            Some(LogError { message: msg }),
        ),

        // A second file save while already building — restart the build.
        // The running child (if any) stays alive until the new build completes.
        (Building { running }, FileChanged(GoChanged)) => (
            Building { running },
            Some(SpawnBuild {
                package: package.to_string(),
                output: binary.clone(),
            }),
        ),

        (Building { running }, FileChanged(AssetChanged)) => (Building { running }, None),

        // ── Stopping ─────────────────────────────────────────────────────────
        (Stopping { binary }, ProcessExited) => (
            WaitingForPort { pid: None, binary: binary.clone() },
            Some(Spawn { binary }),
        ),

        // ── WaitingForPort ───────────────────────────────────────────────────

        // Reactor sends this after spawning the child — store the pid so we can kill it later.
        (WaitingForPort { binary, .. }, ProcessSpawned(pid)) => (
            WaitingForPort { pid: Some(pid), binary },
            None,
        ),

        (WaitingForPort { pid, binary }, PortReady) => (
            Running {
                pid: pid.expect("PortReady before ProcessSpawned — should not happen"),
                binary,
            },
            Some(BroadcastReload),
        ),

        (WaitingForPort { pid, binary }, PortTimeout) => (
            Failed {
                reason: "server did not open port within 10s".to_string(),
                running: pid,
            },
            Some(LogError {
                message: format!(
                    "server did not open port within 10s after spawning {}",
                    binary.display()
                ),
            }),
        ),

        // File changed while waiting — start a new build.
        (WaitingForPort { pid, .. }, FileChanged(GoChanged)) => {
            // Kill-first: kill the waiting-but-not-yet-serving process immediately.
            // Build-first: keep it alive during the build.
            if build_first {
                (
                    Building { running: pid },
                    Some(SpawnBuild {
                        package: package.to_string(),
                        output: binary.clone(),
                    }),
                )
            } else {
                let effect = match pid {
                    Some(pid) => KillAndBuild {
                        pid,
                        package: package.to_string(),
                        output: binary.clone(),
                    },
                    None => SpawnBuild {
                        package: package.to_string(),
                        output: binary.clone(),
                    },
                };
                (Building { running: None }, Some(effect))
            }
        }

        (WaitingForPort { pid, binary }, FileChanged(AssetChanged)) => {
            (WaitingForPort { pid, binary }, None)
        }

        // ── Running ──────────────────────────────────────────────────────────
        (Running { pid, binary }, FileChanged(GoChanged)) => {
            if build_first {
                // Keep old server alive during build; kill it after BuildComplete.
                (
                    Building { running: Some(pid) },
                    Some(SpawnBuild {
                        package: package.to_string(),
                        output: binary.clone(),
                    }),
                )
            } else {
                // Kill-first: kill server now, build immediately.
                (
                    Building { running: None },
                    Some(KillAndBuild {
                        pid,
                        package: package.to_string(),
                        output: binary.clone(),
                    }),
                )
            }
        }

        (Running { pid, binary }, FileChanged(AssetChanged)) => {
            (Running { pid, binary }, Some(BroadcastReload))
        }

        (Running { .. }, ProcessExited) => (
            Failed {
                reason: "server exited unexpectedly".to_string(),
                running: None,
            },
            Some(LogError {
                message: "server exited unexpectedly".to_string(),
            }),
        ),

        // ── Failed ───────────────────────────────────────────────────────────
        (Failed { running, .. }, FileChanged(GoChanged)) => (
            Building { running },
            Some(SpawnBuild {
                package: package.to_string(),
                output: binary.clone(),
            }),
        ),

        (Failed { running, reason }, FileChanged(AssetChanged)) => {
            (Failed { running, reason }, None)
        }

        // ── Catch-all: ignore events that can't happen in the current state ──
        (state, _) => (state, None),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;

    fn bin() -> PathBuf {
        PathBuf::from("/tmp/thunder-test")
    }
    fn pkg() -> &'static str {
        "./examples/counter"
    }

    fn trans(
        state: ProcessState,
        event: WatcherEvent,
        build_first: bool,
    ) -> (ProcessState, Option<SideEffect>) {
        transition(state, event, build_first, pkg(), &bin())
    }

    #[test]
    fn idle_go_change_starts_build() {
        let (next, effect) = trans(ProcessState::Idle, WatcherEvent::FileChanged(FileEvent::GoChanged), false);
        assert!(matches!(next, ProcessState::Building { running: None }));
        assert!(matches!(effect, Some(SideEffect::SpawnBuild { .. })));
    }

    #[test]
    fn idle_asset_change_is_noop() {
        let (next, effect) = trans(ProcessState::Idle, WatcherEvent::FileChanged(FileEvent::AssetChanged), false);
        assert!(matches!(next, ProcessState::Idle));
        assert!(effect.is_none());
    }

    #[test]
    fn build_success_goes_to_stopping() {
        let state = ProcessState::Building { running: None };
        let (next, effect) = trans(state, WatcherEvent::BuildComplete(Ok(bin())), false);
        assert!(matches!(next, ProcessState::Stopping { .. }));
        assert!(matches!(effect, Some(SideEffect::KillAndSpawn { pid: None })));
    }

    #[test]
    fn build_success_build_first_kills_old_pid() {
        let state = ProcessState::Building { running: Some(1234) };
        let (next, effect) = trans(state, WatcherEvent::BuildComplete(Ok(bin())), true);
        assert!(matches!(next, ProcessState::Stopping { .. }));
        assert!(matches!(effect, Some(SideEffect::KillAndSpawn { pid: Some(1234) })));
    }

    #[test]
    fn build_error_goes_to_failed() {
        let state = ProcessState::Building { running: None };
        let (next, effect) = trans(state, WatcherEvent::BuildComplete(Err("syntax error".into())), false);
        assert!(matches!(next, ProcessState::Failed { .. }));
        assert!(matches!(effect, Some(SideEffect::LogError { .. })));
    }

    #[test]
    fn stopping_process_exited_goes_to_waiting() {
        let state = ProcessState::Stopping { binary: bin() };
        let (next, effect) = trans(state, WatcherEvent::ProcessExited, false);
        assert!(matches!(next, ProcessState::WaitingForPort { pid: None, .. }));
        assert!(matches!(effect, Some(SideEffect::Spawn { .. })));
    }

    #[test]
    fn waiting_process_spawned_stores_pid() {
        let state = ProcessState::WaitingForPort { pid: None, binary: bin() };
        let (next, effect) = trans(state, WatcherEvent::ProcessSpawned(9999), false);
        assert!(matches!(next, ProcessState::WaitingForPort { pid: Some(9999), .. }));
        assert!(effect.is_none());
    }

    #[test]
    fn waiting_port_ready_goes_to_running() {
        let state = ProcessState::WaitingForPort { pid: Some(9999), binary: bin() };
        let (next, effect) = trans(state, WatcherEvent::PortReady, false);
        assert!(matches!(next, ProcessState::Running { pid: 9999, .. }));
        assert!(matches!(effect, Some(SideEffect::BroadcastReload)));
    }

    #[test]
    fn running_go_change_kill_first_kills_and_builds() {
        let state = ProcessState::Running { pid: 1234, binary: bin() };
        let (next, effect) = trans(state, WatcherEvent::FileChanged(FileEvent::GoChanged), false);
        assert!(matches!(next, ProcessState::Building { running: None }));
        assert!(matches!(effect, Some(SideEffect::KillAndBuild { pid: 1234, .. })));
    }

    #[test]
    fn running_go_change_build_first_keeps_server() {
        let state = ProcessState::Running { pid: 1234, binary: bin() };
        let (next, effect) = trans(state, WatcherEvent::FileChanged(FileEvent::GoChanged), true);
        assert!(matches!(next, ProcessState::Building { running: Some(1234) }));
        assert!(matches!(effect, Some(SideEffect::SpawnBuild { .. })));
    }

    #[test]
    fn running_asset_change_broadcasts_reload() {
        let state = ProcessState::Running { pid: 1234, binary: bin() };
        let (next, effect) = trans(state, WatcherEvent::FileChanged(FileEvent::AssetChanged), false);
        assert!(matches!(next, ProcessState::Running { .. }));
        assert!(matches!(effect, Some(SideEffect::BroadcastReload)));
    }

    #[test]
    fn running_process_exited_goes_to_failed() {
        let state = ProcessState::Running { pid: 1234, binary: bin() };
        let (next, effect) = trans(state, WatcherEvent::ProcessExited, false);
        assert!(matches!(next, ProcessState::Failed { .. }));
        assert!(matches!(effect, Some(SideEffect::LogError { .. })));
    }

    #[test]
    fn failed_go_change_restarts_build() {
        let state = ProcessState::Failed { reason: "boom".into(), running: None };
        let (next, effect) = trans(state, WatcherEvent::FileChanged(FileEvent::GoChanged), false);
        assert!(matches!(next, ProcessState::Building { running: None }));
        assert!(matches!(effect, Some(SideEffect::SpawnBuild { .. })));
    }
}
