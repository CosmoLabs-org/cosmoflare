---
ulid: 01M25W8AFZB4493Q9063MGMDH4
title: Consolidate isNotFoundCF and isNotFoundError into one structured detector
created: "2026-09-10T18:41:23.967639+04:00"
status: seed
source: agent
origin:
    session: 49
    trigger: session-end simplify review (reuse finding 3)
---

# Consolidate isNotFoundCF and isNotFoundError into one structured detector

pkg/cosmoflare now has two not-found detectors: isNotFoundCF (ratelimit.go:209, structured errors.As) and isNotFoundError (cors.go:418, lowercase-message matching that can false-positive). Upgrade cors.go to the structured check and keep one symbol. surfaced by /simplify pass 2026-09-10.
