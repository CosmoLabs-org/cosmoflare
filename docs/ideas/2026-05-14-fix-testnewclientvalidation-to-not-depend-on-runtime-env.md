---
id: IDEA-020
title: Fix TestNewClientValidation to not depend on runtime env vars
created: "2026-05-14T23:56:59.808337-03:00"
status: harvested
source: human
origin:
    session: 2027
---




# Fix TestNewClientValidation to not depend on runtime env vars

Test fails when CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN are set in the shell. Should use t.Setenv or clear env in test setup.
