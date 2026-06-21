---
id: IDEA-036
title: Untrack build/r2go2 (38MB binary tracked inside a gitignored build/)
created: "2026-06-20T21:05:23.090769-03:00"
status: seed
source: human
origin:
    session: 17
---

# Untrack build/r2go2 (38MB binary tracked inside a gitignored build/)

build/ is gitignored but build/r2go2 is an individually-tracked 38MB binary, so every rebuild commits a new 38MB blob into history (saw 36533282->38125330 bytes this session). build/cosmoflare is correctly untracked. Either git rm --cached build/r2go2 (fully ignore build/) if no release process depends on the tracked binary, or move release binaries to a release-artifacts mechanism. Surfaced during session-end commit of the keychain-fixed rebuild.
