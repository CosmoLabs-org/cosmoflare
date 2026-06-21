// Minimal app shell. G-06 (P-06) replaces this with the full shell:
// Header (account switcher + health dots) + Sidebar + routed main pane, wired
// to the daemon's REST/SSE API. Kept trivial here so the scaffold builds and
// `tauri dev` launches a window before the real UI lands.
export default function App() {
  return (
    <main>
      <h1>Cosmoflare</h1>
      <p>Desktop dashboard — UI under construction (see G-06).</p>
    </main>
  );
}
