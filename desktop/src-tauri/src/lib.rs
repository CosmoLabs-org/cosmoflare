// Tauri v2 application entrypoint. G-04 scaffolds the minimal builder; G-05
// (P-05) adds the daemon lifecycle here: spawn the cosmoflare sidecar, parse
// the handshake, health-poll, kill on exit, and expose a `daemon_endpoint`
// command to the webview.
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .run(tauri::generate_context!())
        .expect("error while running cosmoflare desktop app");
}
