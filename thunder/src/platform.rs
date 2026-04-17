use std::io;
use std::path::{Path, PathBuf};
use std::process::Child;

/// Abstracts OS-specific concerns: process signaling, binary naming, and binary swapping.
/// Everything else in the codebase uses this trait — no `#[cfg]` outside this file.
pub trait Platform: Send + Sync {
    /// Send a graceful shutdown signal to a Child. Unix: SIGTERM. Windows: Ctrl+Break.
    fn terminate(&self, child: &mut Child) -> io::Result<()>;

    /// Force-kill a Child with no grace period. Unix: SIGKILL. Windows: TerminateProcess.
    fn force_kill(&self, child: &mut Child) -> io::Result<()>;

    /// Send a graceful shutdown signal by PID (no Child handle required).
    fn terminate_pid(&self, pid: u32) -> io::Result<()>;

    /// Force-kill by PID (no Child handle required).
    fn force_kill_pid(&self, pid: u32) -> io::Result<()>;

    /// Return the platform-correct binary path for a given base path.
    /// Unix: unchanged. Windows: appends `.exe`.
    fn binary_path(&self, base: &Path) -> PathBuf;

    /// Swap old binary for new binary atomically where possible.
    /// Unix: `rename(new, target)` — atomic on the same filesystem.
    /// Windows: copy new → target, delete new (cannot rename a running `.exe`).
    fn swap_binary(&self, new: &Path, target: &Path) -> io::Result<()>;
}

// ── Unix ────────────────────────────────────────────────────────────────────

#[cfg(unix)]
pub struct Unix;

#[cfg(unix)]
impl Platform for Unix {
    fn terminate(&self, child: &mut Child) -> io::Result<()> {
        self.terminate_pid(child.id())
    }

    fn force_kill(&self, child: &mut Child) -> io::Result<()> {
        child.kill()
    }

    fn terminate_pid(&self, pid: u32) -> io::Result<()> {
        // SAFETY: pid is a valid PID for a process we own.
        // The risk is a wrong PID (logic error), not a memory-safety issue.
        unsafe { libc::kill(pid as i32, libc::SIGTERM) };
        Ok(())
    }

    fn force_kill_pid(&self, pid: u32) -> io::Result<()> {
        // SAFETY: same as terminate_pid.
        unsafe { libc::kill(pid as i32, libc::SIGKILL) };
        Ok(())
    }

    fn binary_path(&self, base: &Path) -> PathBuf {
        base.to_path_buf()
    }

    fn swap_binary(&self, new: &Path, target: &Path) -> io::Result<()> {
        std::fs::rename(new, target)
    }
}

// ── Windows (stub — compiles, not yet functional) ───────────────────────────

#[cfg(windows)]
pub struct Windows;

#[cfg(windows)]
impl Platform for Windows {
    fn terminate(&self, child: &mut Child) -> io::Result<()> {
        self.terminate_pid(child.id())
    }

    fn force_kill(&self, child: &mut Child) -> io::Result<()> {
        child.kill()
    }

    fn terminate_pid(&self, _pid: u32) -> io::Result<()> {
        // TODO: GenerateConsoleCtrlEvent(CTRL_BREAK_EVENT, pid)
        // Requires windows-sys crate. Falls back to force-kill until implemented.
        Ok(())
    }

    fn force_kill_pid(&self, _pid: u32) -> io::Result<()> {
        // TODO: OpenProcess + TerminateProcess
        Ok(())
    }

    fn binary_path(&self, base: &Path) -> PathBuf {
        base.with_extension("exe")
    }

    fn swap_binary(&self, new: &Path, target: &Path) -> io::Result<()> {
        // Windows locks the .exe inode, not the filename — copy+delete works.
        // TODO: verify experimentally that this succeeds while old binary is running.
        std::fs::copy(new, target)?;
        std::fs::remove_file(new)?;
        Ok(())
    }
}

// ── Factory ─────────────────────────────────────────────────────────────────

/// Returns the correct platform implementation for the current OS.
/// Returns `Arc<dyn Platform>` so it can be shared across spawned tasks.
pub fn current_platform() -> std::sync::Arc<dyn Platform> {
    #[cfg(unix)]
    return std::sync::Arc::new(Unix);

    #[cfg(windows)]
    return std::sync::Arc::new(Windows);
}
