use std::path::PathBuf;

/// Classifies which kind of file changed, determining the reactor's response.
#[derive(Debug, Clone)]
pub enum FileEvent {
    /// A `.go` file changed — triggers a rebuild and server restart.
    GoChanged,
    /// A `.html`, `.css`, or `.js` file changed — triggers a browser reload only.
    AssetChanged,
}

/// The single channel type for the reactor loop.
/// Every event source — file watcher thread and spawned async tasks — sends this.
#[derive(Debug)]
pub enum WatcherEvent {
    // External — from the file watcher thread
    FileChanged(FileEvent),

    // Internal — from tasks spawned by the reactor
    /// `Ok(binary_path)` on success, `Err(stderr)` on build failure.
    BuildComplete(Result<PathBuf, String>),
    /// The Go server's port accepted a TCP connection — server is ready.
    PortReady,
    /// Port poll timed out (10s) — server failed to come up.
    PortTimeout,
    /// The child process exited (expected during restart, or unexpected crash).
    ProcessExited,
    /// PID of the newly spawned server process, sent back by execute_effect(Spawn).
    /// The state machine stores this so it can kill the right process later.
    ProcessSpawned(u32),
}

/// What the state machine returns instead of performing I/O directly.
/// The reactor executes these — the state machine has no side effects.
#[derive(Debug)]
pub enum SideEffect {
    /// Start a `go build`. Used on the first file change and in build-first mode.
    SpawnBuild {
        package: String,
        output: PathBuf,
    },
    /// Kill-first: kill the running server by PID, then immediately start a build.
    /// Used when a GoChanged event fires while a server is already running.
    KillAndBuild {
        pid: u32,
        package: String,
        output: PathBuf,
    },
    /// Kill old server (if any) after a successful build, then signal ready to spawn.
    /// pid: None in kill-first (server was already killed by KillAndBuild).
    ///      Some in build-first (server stayed alive during the build).
    KillAndSpawn {
        pid: Option<u32>,
    },
    /// Spawn the compiled binary and start monitoring it.
    Spawn {
        binary: PathBuf,
    },
    BroadcastReload,
    LogError {
        message: String,
    },
}
