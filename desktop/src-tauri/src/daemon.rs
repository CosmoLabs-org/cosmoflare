//! Daemon lifecycle: the Rust shell spawns the bundled `cosmoflare` sidecar,
//! parses its stdout handshake, health-polls it, surfaces the endpoint to the
//! webview, and kills it on app exit. The pure pieces (handshake parse, backoff
//! schedule, health-poll) are unit/integration-tested; the Tauri shell-plugin
//! spawn is a thin wrapper over the same logic.
//!
//! Account/credentials are resolved lazily by the Go daemon per request. The
//! shell sets COSMOFLARE_NO_KEYCHAIN=1 on the child so it never probes the OS
//! keychain (BR-04).

use serde::{Deserialize, Serialize};
use std::sync::Mutex;
use std::time::{Duration, Instant};
use tauri::Manager;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tauri_plugin_shell::ShellExt;

/// How long to wait for the handshake line + for /healthz to go ready.
const HANDSHAKE_TIMEOUT: Duration = Duration::from_secs(10);
/// /healthz polling interval while waiting for the daemon to come up.
const HEALTH_POLL_INTERVAL: Duration = Duration::from_millis(200);

/// The one-line JSON the Go daemon prints to stdout on startup.
/// `{ "addr": "127.0.0.1:<port>", "token": "<random>" }`
#[derive(Debug, Clone, Deserialize, Serialize, PartialEq, Eq)]
pub struct Handshake {
    pub addr: String,
    pub token: String,
}

/// Parse the handshake JSON line. Trims whitespace/newlines defensively.
pub fn parse_handshake(line: &str) -> Result<Handshake, serde_json::Error> {
    serde_json::from_str(line.trim())
}

/// The daemon endpoint surfaced to the webview via the `daemon_endpoint`
/// command. The webview uses `url` + `token` (Bearer) for REST + SSE.
#[derive(Debug, Clone, Serialize)]
pub struct DaemonEndpoint {
    pub url: String,
    pub token: String,
}

/// Exponential-backoff schedule for the spawn watchdog (3 attempts), per BR-02
/// / the plan's "3× exponential backoff, then surface an error screen".
pub fn backoff_delays() -> [Duration; 3] {
    [
        Duration::from_millis(500),
        Duration::from_secs(1),
        Duration::from_secs(2),
    ]
}

/// Health-poll failure reason.
#[derive(Debug)]
pub enum WaitError {
    /// The deadline elapsed before /healthz returned 200.
    Timeout,
    /// /healthz answered but not 200 (e.g. wrong token / daemon erroring).
    Http(String),
}

/// Poll `GET /healthz` with the bearer token until it returns 200 or `timeout`
/// elapses. Connection-refused (daemon still starting) is retried; any non-200
/// that persists past the deadline is an Http error.
pub fn wait_until_ready(addr: &str, token: &str, timeout: Duration) -> Result<(), WaitError> {
    let url = format!("http://{}/healthz", addr);
    let auth = format!("Bearer {}", token);
    let deadline = Instant::now() + timeout;
    loop {
        match ureq::get(&url).set("Authorization", &auth).call() {
            Ok(resp) if resp.status() == 200 => return Ok(()),
            Ok(resp) => {
                if Instant::now() >= deadline {
                    return Err(WaitError::Http(format!("status {}", resp.status())));
                }
            }
            Err(ureq::Error::Status(code, _)) => {
                if Instant::now() >= deadline {
                    return Err(WaitError::Http(format!("status {}", code)));
                }
            }
            Err(_) => {
                // Connection refused / reset while the daemon boots — retry.
                if Instant::now() >= deadline {
                    return Err(WaitError::Timeout);
                }
            }
        }
        std::thread::sleep(HEALTH_POLL_INTERVAL);
    }
}

/// Spawn-time failure. Payloads are surfaced via Debug logging only.
#[derive(Debug)]
#[allow(dead_code)]
enum SpawnError {
    Sidecar(String),
    Spawn(String),
    HandshakeTimeout,
    Health,
}

/// Shared state holding the live child (so it can be killed on exit) and the
/// resolved endpoint (served to the webview).
#[derive(Default)]
pub struct DaemonState {
    endpoint: Mutex<Option<DaemonEndpoint>>,
    child: Mutex<Option<CommandChild>>,
}

impl DaemonState {
    fn set_endpoint(&self, ep: DaemonEndpoint) {
        *self.endpoint.lock().unwrap() = Some(ep);
    }

    fn set_child(&self, child: CommandChild) {
        // Replace any previous child, killing it first (watchdog re-spawn).
        if let Some(prev) = self.child.lock().unwrap().take() {
            let _ = prev.kill();
        }
        *self.child.lock().unwrap() = Some(child);
    }

    /// Kill the daemon child. Called on window close / app quit.
    pub fn kill_child(&self) {
        if let Some(child) = self.child.lock().unwrap().take() {
            let _ = child.kill();
        }
    }
}

/// Tauri command: return the daemon endpoint to the webview (None until the
/// handshake completes — the webview polls or waits on a `status` SSE frame).
#[tauri::command]
pub fn daemon_endpoint(state: tauri::State<'_, DaemonState>) -> Option<DaemonEndpoint> {
    state.endpoint.lock().unwrap().clone()
}

/// Start the daemon lifecycle. Spawns an async task that retries the
/// spawn→handshake→health sequence on the backoff schedule (3 attempts), then
/// stores the endpoint for the webview. After the final failed attempt it
/// logs and leaves the endpoint unset (the UI surfaces a setup/error screen).
pub fn start_daemon(handle: tauri::AppHandle) {
    tauri::async_runtime::spawn(async move {
        let state = handle.state::<DaemonState>();
        let delays = backoff_delays();
        for (i, &delay) in delays.iter().enumerate() {
            match spawn_and_handshake(&handle).await {
                Ok(ep) => {
                    state.set_endpoint(ep);
                    println!("[cosmoflare] daemon ready");
                    return;
                }
                Err(e) => {
                    eprintln!(
                        "[cosmoflare] daemon start attempt {}/{} failed: {:?}",
                        i + 1,
                        delays.len(),
                        e
                    );
                    if i + 1 < delays.len() {
                        tokio::time::sleep(delay).await;
                    }
                }
            }
        }
        eprintln!(
            "[cosmoflare] daemon failed to start after {} attempts",
            delays.len()
        );
    });
}

/// Spawn the sidecar, read the handshake from stdout, and health-poll. The
/// child is stored in state immediately so it can always be killed.
async fn spawn_and_handshake(handle: &tauri::AppHandle) -> Result<DaemonEndpoint, SpawnError> {
    let (mut rx, child) = handle
        .shell()
        .sidecar("cosmoflare")
        .map_err(|e| SpawnError::Sidecar(e.to_string()))?
        .args(["serve", "--addr", "127.0.0.1:0"])
        .env("COSMOFLARE_NO_KEYCHAIN", "1")
        .spawn()
        .map_err(|e| SpawnError::Spawn(e.to_string()))?;

    handle.state::<DaemonState>().set_child(child);

    let deadline = Instant::now() + HANDSHAKE_TIMEOUT;
    while Instant::now() < deadline {
        match rx.recv().await {
            Some(CommandEvent::Stdout(bytes)) => {
                let line = String::from_utf8_lossy(&bytes);
                if let Ok(hs) = parse_handshake(&line) {
                    let (addr, token) = (hs.addr.clone(), hs.token.clone());
                    // Blocking health-poll off the async executor.
                    let ready =
                        tauri::async_runtime::spawn_blocking(move || {
                            wait_until_ready(&addr, &token, HANDSHAKE_TIMEOUT)
                        })
                        .await;
                    match ready {
                        Ok(Ok(())) => {
                            return Ok(DaemonEndpoint {
                                url: format!("http://{}", hs.addr),
                                token: hs.token,
                            })
                        }
                        _ => {
                            // Daemon handshook but never went healthy — kill it
                            // so a hung process can't outlive the attempt.
                            handle.state::<DaemonState>().kill_child();
                            return Err(SpawnError::Health);
                        }
                    }
                }
            }
            Some(other) => {
                eprintln!("[cosmoflare] sidecar event: {:?}", other);
            }
            None => break,
        }
    }
    // No handshake line within the deadline — kill the silent child.
    handle.state::<DaemonState>().kill_child();
    Err(SpawnError::HandshakeTimeout)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_handshake_line() {
        let line = r#"{"addr":"127.0.0.1:54123","token":"abc"}"#;
        let h = parse_handshake(line).unwrap();
        assert_eq!(h.addr, "127.0.0.1:54123");
        assert_eq!(h.token, "abc");
    }

    #[test]
    fn parses_handshake_with_trailing_whitespace() {
        let h = parse_handshake(r#"{"addr":"127.0.0.1:1","token":"t"}  "#).unwrap();
        assert_eq!(h.addr, "127.0.0.1:1");
    }

    #[test]
    fn rejects_malformed_handshake() {
        assert!(parse_handshake("not json").is_err());
        assert!(parse_handshake(r#"{"addr":"x"}"#).is_err()); // missing token
    }

    #[test]
    fn backoff_schedule_is_exponential_and_has_three_attempts() {
        let d = backoff_delays();
        assert_eq!(d.len(), 3);
        assert!(d[0] < d[1]);
        assert!(d[1] < d[2]);
    }

    #[test]
    fn wait_until_ready_times_out_on_unreachable_address() {
        // Port 1 is reserved/unroutable — the poll must time out, not hang.
        let err = wait_until_ready("127.0.0.1:1", "t", Duration::from_millis(300));
        assert!(matches!(err, Err(WaitError::Timeout)), "got {:?}", err);
    }
}
