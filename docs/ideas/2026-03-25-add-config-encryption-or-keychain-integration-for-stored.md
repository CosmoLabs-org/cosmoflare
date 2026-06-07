---
id: IDEA-033
title: Add config encryption or keychain integration for stored credentials
created: "2026-03-25T03:57:40.621742+01:00"
status: withered
source: agent
origin:
    session: 17
    trigger: audit-infrastructure-security
tags:
    - audit
    - security
promoted_to: FEAT-005
resolution:
    reason: implemented
    date: "2026-06-07T01:26:32.850881-03:00"
    ref: FEAT-005
    note: Resolved via FEAT-005 (implemented)
---

# Add config encryption or keychain integration for stored credentials

# Add config encryption or keychain integration for stored credentials

# Add config encryption or keychain integration for stored credentials

Config at ~/.r2go2/config.yaml stores API tokens and secret keys in plaintext. While 0600 permissions help, this is insufficient for credential security. Consider macOS Keychain via go-keyring, Linux secret-service, or encrypted config with master passphrase.
