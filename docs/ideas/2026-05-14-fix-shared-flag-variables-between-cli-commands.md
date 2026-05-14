---
id: IDEA-017
title: Fix shared flag variables between CLI commands
created: "2026-05-14T05:28:07.444745-03:00"
status: seed
source: human
origin:
    session: 2027
---

# Fix shared flag variables between CLI commands

cmd/kv.go:153 reuses workerForce for KV --force flag. workerCompatDate shared between deploy and settings subcommands. These are shared state bugs — flag value from one subcommand can leak to another. Fix: declare local variables in each command's init() or use unique names like kvForce, settingsCompatDate.
