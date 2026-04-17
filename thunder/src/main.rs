mod cli;
mod event;
mod platform;
mod process;
mod state;
mod watcher;
mod ws_server;

use std::sync::Arc;

use clap::Parser;
use cli::{Cli, Command};
use event::{SideEffect, WatcherEvent};
use state::{ProcessState, transition};
use tokio::sync::mpsc;

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "info".into()),
        )
        .init();

    let cli = Cli::parse();

    match cli.command {
        Command::Watch(args) => {
            let watch_dir = args.resolved_watch_dir();
            let extensions = args.extensions();
            let ws_port = args.ws_port;
            let build_first = args.build_first;
            let package = args.go_package.clone();

            let platform = platform::current_platform();
            let binary = process::temp_binary_path(&package, platform.as_ref());

            // Unified reactor channel — all event sources send WatcherEvent here.
            let (tx, mut rx) = mpsc::channel::<WatcherEvent>(64);

            // Start WebSocket server for live-reload.
            let ws_tx = ws_server::start(ws_port).await;

            // Start file watcher on a background thread.
            // Bridges std::sync::mpsc (watcher thread) to tokio mpsc (reactor).
            let tx_for_watcher = {
                let (std_tx, std_rx) = std::sync::mpsc::channel::<WatcherEvent>();
                let tx_clone = tx.clone();
                std::thread::spawn(move || {
                    while let Ok(event) = std_rx.recv() {
                        let _ = tx_clone.blocking_send(event);
                    }
                });
                std_tx
            };
            watcher::start(&watch_dir, extensions, tx_for_watcher);

            tracing::info!(
                package = %package,
                watch_dir = %watch_dir.display(),
                ws_port = ws_port,
                build_first = build_first,
                "thunder watch starting"
            );

            // Start in Building state so BuildComplete is handled correctly on startup.
            let mut state = ProcessState::Building { running: None };
            let effect = SideEffect::SpawnBuild {
                package: package.clone(),
                output: binary.clone(),
            };
            execute_effect(effect, &tx, &ws_tx, &platform, ws_port).await;

            // Reactor loop — exits on Ctrl+C or channel close.
            loop {
                tokio::select! {
                    event = rx.recv() => {
                        let Some(event) = event else { break };
                        let (new_state, effect) = transition(state, event, build_first, &package, &binary);
                        state = new_state;
                        if let Some(effect) = effect {
                            execute_effect(effect, &tx, &ws_tx, &platform, ws_port).await;
                        }
                    }
                    _ = tokio::signal::ctrl_c() => {
                        tracing::info!("shutting down...");
                        break;
                    }
                }
            }

            // Cleanup: kill the running server and delete the temp binary.
            let running_pid = match &state {
                ProcessState::Running { pid, .. } => Some(*pid),
                ProcessState::WaitingForPort { pid: Some(pid), .. } => Some(*pid),
                _ => None,
            };
            if let Some(pid) = running_pid {
                tracing::info!(%pid, "stopping server");
                let p = Arc::clone(&platform);
                tokio::task::spawn_blocking(move || process::kill_by_pid(pid, p.as_ref()))
                    .await
                    .ok();
            }
            let _ = std::fs::remove_file(&binary);
            tracing::info!("done");
        }
    }
}

/// Executes a side effect as a non-blocking task.
/// Each spawned task sends its result back into the reactor channel as a WatcherEvent.
async fn execute_effect(
    effect: SideEffect,
    tx: &mpsc::Sender<WatcherEvent>,
    ws_tx: &tokio::sync::broadcast::Sender<()>,
    platform: &Arc<dyn platform::Platform>,
    ws_port: u16,
) {
    match effect {
        // ── Build ─────────────────────────────────────────────────────────────
        SideEffect::SpawnBuild { package, output } => {
            let tx = tx.clone();
            tokio::task::spawn_blocking(move || {
                tracing::info!(%package, "building...");
                match process::build(&package, &output) {
                    Ok(()) => {
                        tracing::info!("build succeeded");
                        let _ = tx.blocking_send(WatcherEvent::BuildComplete(Ok(output)));
                    }
                    Err(stderr) => {
                        let _ = tx.blocking_send(WatcherEvent::BuildComplete(Err(stderr)));
                    }
                }
            });
        }

        // ── Kill then build (kill-first GoChanged) ────────────────────────────
        SideEffect::KillAndBuild { pid, package, output } => {
            let tx = tx.clone();
            let platform = Arc::clone(platform);
            tokio::task::spawn_blocking(move || {
                tracing::info!(%pid, "stopping server before rebuild...");
                process::kill_by_pid(pid, platform.as_ref());
                tracing::info!(%package, "building...");
                match process::build(&package, &output) {
                    Ok(()) => {
                        tracing::info!("build succeeded");
                        let _ = tx.blocking_send(WatcherEvent::BuildComplete(Ok(output)));
                    }
                    Err(stderr) => {
                        let _ = tx.blocking_send(WatcherEvent::BuildComplete(Err(stderr)));
                    }
                }
            });
        }

        // ── Kill old server after build-first build ───────────────────────────
        SideEffect::KillAndSpawn { pid } => {
            let tx = tx.clone();
            let platform = Arc::clone(platform);
            tokio::task::spawn_blocking(move || {
                if let Some(pid) = pid {
                    tracing::info!(%pid, "stopping old server after build...");
                    process::kill_by_pid(pid, platform.as_ref());
                }
                let _ = tx.blocking_send(WatcherEvent::ProcessExited);
            });
        }

        // ── Spawn new server ─────────────────────────────────────────────────
        SideEffect::Spawn { binary } => {
            let tx = tx.clone();
            match process::spawn(&binary, ws_port) {
                Ok(mut child) => {
                    // Capture pid BEFORE moving child into the monitor task.
                    let pid = child.id();
                    tracing::info!(%pid, binary = %binary.display(), "server spawned");

                    // Read stdout on a plain thread — extracts port, forwards logs.
                    let (port_tx, port_rx) = std::sync::mpsc::sync_channel::<u16>(1);
                    if let Some(stdout) = child.stdout.take() {
                        process::read_stdout(stdout, port_tx);
                    }

                    // Monitor for unexpected exit. Child is moved here for wait().
                    let tx_exit = tx.clone();
                    tokio::task::spawn_blocking(move || {
                        let _ = child.wait(); // reaps zombie
                        let _ = tx_exit.blocking_send(WatcherEvent::ProcessExited);
                    });

                    // Send pid back to the reactor so the state machine can store it.
                    let _ = tx.send(WatcherEvent::ProcessSpawned(pid)).await;

                    // Start port polling once we know the port from stdout.
                    let tx_port = tx.clone();
                    tokio::spawn(async move {
                        let port = tokio::task::spawn_blocking(move || {
                            port_rx
                                .recv_timeout(std::time::Duration::from_secs(5))
                                .ok()
                        })
                        .await
                        .ok()
                        .flatten();

                        if let Some(port) = port {
                            process::poll_port(port, tx_port).await;
                        } else {
                            tracing::warn!("did not detect port from server stdout — skipping port poll");
                        }
                    });
                }
                Err(e) => {
                    let _ = tx
                        .send(WatcherEvent::BuildComplete(Err(format!(
                            "failed to spawn server: {e}"
                        ))))
                        .await;
                }
            }
        }

        SideEffect::BroadcastReload => {
            let _ = ws_tx.send(());
            tracing::info!("browser reload signal sent");
        }

        SideEffect::LogError { message } => {
            tracing::error!("{message}");
        }
    }
}
