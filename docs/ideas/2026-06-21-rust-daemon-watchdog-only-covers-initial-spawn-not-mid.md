---
id: IDEA-038
title: Rust daemon watchdog only covers initial spawn, not mid-session crash
created: "2026-06-21T03:02:18.099971-03:00"
status: seed
source: agent
origin:
    session: 18
    trigger: worktree-end reflection
    file: desktop/src-tauri/src/daemon.rs
tags:
    - desktop
    - rust
    - reliability
---

# Rust daemon watchdog only covers initial spawn, not mid-session crash

start_daemon retries the spawn->handshake->health sequence 3x with backoff, but if the daemon crashes AFTER a successful startup (mid-session), the CommandEvent::Terminated is only logged, not acted on. BR-02 calls for a runtime watchdog. A full watchdog would monitor Terminated and re-spawn (with the same backoff + endpoint re-resolution). Edge case for v1, but worth a follow-up.
