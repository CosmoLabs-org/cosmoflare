// Tauri v2 application entrypoint. Owns the cosmoflare daemon lifecycle:
// spawns the bundled sidecar on startup, health-polls it, surfaces its endpoint
// to the webview, and kills it on app exit (window close AND Cmd-Q/quit). See
// `daemon.rs`.

pub mod daemon;

use daemon::{daemon_endpoint, start_daemon, DaemonState};
use tauri::Manager;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .manage(DaemonState::default())
        .setup(|app| {
            start_daemon(app.handle().clone());
            Ok(())
        })
        .on_window_event(|window, event| {
            // Window close (red button / non-macOS last-window) → stop the daemon.
            if let tauri::WindowEvent::CloseRequested { .. } = event {
                if let Some(state) = window.app_handle().try_state::<DaemonState>() {
                    state.kill_child();
                }
            }
        })
        .invoke_handler(tauri::generate_handler![daemon_endpoint])
        .build(tauri::generate_context!())
        .expect("error while building cosmoflare desktop app")
        .run(|handle, event| {
            // App-level exit (Cmd-Q, dock quit, ExitRequested) does NOT fire
            // WindowEvent::CloseRequested, so the daemon must be killed here too.
            // Without this the cosmoflare sidecar is orphaned — a leaked process
            // holding a localhost port — on the most common quit paths.
            // kill_child is idempotent, so overlap with the window handler is safe.
            if matches!(
                event,
                tauri::RunEvent::ExitRequested { .. } | tauri::RunEvent::Exit
            ) {
                if let Some(state) = handle.try_state::<DaemonState>() {
                    state.kill_child();
                }
            }
        });
}
