// React entry (P-03 shell swap). The vanilla entry (src/main.ts) stays in
// the tree as reference until P-06 completes; nothing imports it anymore.
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "../styles.css";

// The push service worker registration is part of the app bootstrap.
if ("serviceWorker" in navigator) {
  navigator.serviceWorker.register("/sw.js").catch((err: unknown) => {
    console.error("pager: service worker registration failed", err);
  });
}

const rootEl = document.getElementById("app");
if (rootEl) {
  createRoot(rootEl).render(
    <StrictMode>
      <App />
    </StrictMode>,
  );
}
