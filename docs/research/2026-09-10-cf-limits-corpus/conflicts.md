# Limits corpus — conflicts register (2026-10-08 synthesis)

Sources: qwen-results.md (inline JSON + table) x gemini pricing-catalog-draft.json. Grok results absent — two-way comparison only.

## Value conflicts (0)

None — all shared ids agree on free-tier values.

## Coverage notes

- Catalog entries: 66 (qwen JSON 66 + table-only 0).
- Gemini-only ids (pricing without a limit row, informational): 51: workers.inbound_requests, workers.base_subscription, workers.data_egress, workers.build_minutes, workers.concurrent_builds, r2.standard_storage, r2.standard_class_a_operations, r2.standard_class_b_operations.
- Every entry carries source_url + verified_on (schema v1 / IMP-032).
