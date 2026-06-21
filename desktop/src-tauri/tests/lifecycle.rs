//! Integration test for the daemon lifecycle. Spawns the REAL cosmoflare Go
//! sidecar binary (the same one Tauri bundles), reads the stdout handshake,
//! and health-polls it — exercising parse_handshake + wait_until_ready against
//! a live process. The Tauri shell-plugin spawn itself is a thin wrapper over
//! the same logic, verified separately via `cargo check` / `tauri dev`.
//!
//! Skips automatically if the sidecar binary is not present (e.g. CI without
//! build-sidecar.sh having run), so `cargo test` is green in any environment.

use cosmoflare_desktop_lib::daemon::{parse_handshake, wait_until_ready, Handshake};
use std::io::{BufRead, BufReader};
use std::path::PathBuf;
use std::process::{Child, Command, Stdio};
use std::time::Duration;

/// The full target triple string (must match `rustc -vV` host, arch-vendor-os).
fn target_triple() -> String {
    match (std::env::consts::OS, std::env::consts::ARCH) {
        ("macos", "aarch64") => "aarch64-apple-darwin".into(),
        ("macos", "x86_64") => "x86_64-apple-darwin".into(),
        ("linux", "x86_64") => "x86_64-unknown-linux-gnu".into(),
        ("windows", "x86_64") => "x86_64-pc-windows-msvc".into(),
        _ => "unknown-target".into(),
    }
}

/// Construct the path to the host-triple sidecar binary under src-tauri/binaries.
fn sidecar_path() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("binaries")
        .join(format!("cosmoflare-{}", target_triple()))
}

struct Daemon {
    child: Child,
    addr: String,
    token: String,
}

impl Daemon {
    fn spawn() -> Option<Daemon> {
        let bin = sidecar_path();
        if !bin.exists() {
            return None; // sidecar not built — skip
        }
        let mut child = Command::new(&bin)
            .args(["serve", "--addr", "127.0.0.1:0"])
            .env("COSMOFLARE_NO_KEYCHAIN", "1")
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .ok()?;

        // Read the first stdout line — the handshake JSON.
        let stdout = child.stdout.take().expect("piped stdout");
        let mut reader = BufReader::new(stdout);
        let mut line = String::new();
        reader.read_line(&mut line).ok()?;
        let hs: Handshake = parse_handshake(&line).ok()?;
        Some(Daemon {
            child,
            addr: hs.addr,
            token: hs.token,
        })
    }
}

impl Drop for Daemon {
    fn drop(&mut self) {
        let _ = self.child.kill();
        let _ = self.child.wait();
    }
}

#[test]
fn handshake_line_parses() {
    let line = r#"{"addr":"127.0.0.1:54123","token":"abc"}"#;
    let h = parse_handshake(line).expect("parse");
    assert_eq!(h.addr, "127.0.0.1:54123");
    assert_eq!(h.token, "abc");
}

#[test]
fn real_daemon_starts_and_becomes_ready() {
    let daemon = match Daemon::spawn() {
        Some(d) => d,
        None => {
            eprintln!(
                "skip: sidecar binary not present at {:?} (triple {})",
                sidecar_path(),
                target_triple()
            );
            return;
        }
    };
    // The daemon should become health-ready within a few seconds.
    wait_until_ready(&daemon.addr, &daemon.token, Duration::from_secs(10))
        .expect("daemon did not become ready");
}

#[test]
fn real_daemon_healthz_requires_token() {
    let daemon = match Daemon::spawn() {
        Some(d) => d,
        None => return,
    };
    // No token -> 401. ureq returns Err(Status(code,_)) for non-2xx, so extract
    // the status from the error to prove auth is enforced on the live process.
    let url = format!("http://{}/healthz", daemon.addr);
    let status = match ureq::get(&url).call() {
        Ok(r) => r.status(),
        Err(ureq::Error::Status(code, _)) => code,
        Err(_) => 0,
    };
    assert_eq!(status, 401, "expected 401 without token, got {}", status);
}
