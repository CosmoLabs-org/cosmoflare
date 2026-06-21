// Tauri v2 application entrypoint. Owns the cosmoflare daemon lifecycle:
// spawns the bundled sidecar on startup, health-polls it, surfaces its endpoint
// to the webview, and kills it on window close. See `daemon.rs`.

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
            // Kill the daemon when the main window closes / the app quits.
            if let tauri::WindowEvent::CloseRequested { .. } = event {
                if let Some(state) = window.app_handle().try_state::<DaemonState>() {
                    state.kill_child();
                }
            }
        })
        .invoke_handler(tauri::generate_handler![daemon_endpoint])
        .run(tauri::generate_context!())
        .expect("error while running cosmoflare desktop app");
}
