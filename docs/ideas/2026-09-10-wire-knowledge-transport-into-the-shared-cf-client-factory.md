---
ulid: 01M25W87Z6A8QTFXE9ZQFQQZ1A
title: Wire knowledge.Transport into the shared CF client factory
created: "2026-09-10T18:41:21.382929+04:00"
status: seed
source: agent
origin:
    session: 49
    trigger: session-end simplify review (altitude finding 1)
---

# Wire knowledge.Transport into the shared CF client factory

# Wire knowledge.Transport into the shared CF client factory

The knowledge Transport is installed only in NewRateLimitServiceFromCreds; ~24 other cloudflare.NewWithAPIToken sites get no transport. The pack design (pass-through outside scopes) makes it globally safe. Install at client.go:117 (wrap cfg.httpClient) or add a newCFAPI(apiToken) helper all FromCreds constructors call. FEAT-012 deliberately deferred this; revisit when a second pack arrives.

---

Triage 2026-09-16 (session 58): HOLD confirmed — wiring is safe (pass-through design) but user value is gated on the second knowledge pack; with one pack only ratelimit-scope errors decode at the ~24 unwired sites. Revisit when pack #2 lands.
