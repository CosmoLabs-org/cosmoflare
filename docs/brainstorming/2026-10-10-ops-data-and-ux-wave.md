# Ops data + UX wave — brainstorm (2026-10-10)

Operator asks, streamed live during session 2034's tail. Every API claim
below was verified live against the account on 2026-10-10.

## What was asked

1. Show more account data: emails sent, the full list of domain names with
   their expirations and statuses.
2. A section just for projects and their consumption across Cloudflare.
3. Standard sorting on tables (top-right controls, largest numbers first by
   default, changeable order).
4. Progress bars, micro-animations, and reusable components; better section
   headings and search/filter bars.
5. (Shipped same session: pairing page redesign, overview plan line with
   Workers Paid $5.00/mo + overage.)

## Verified data facts (2026-10-10)

| Need | Source | Status with current token |
|---|---|---|
| Domains + expiry | `GET /zones` → name, status, `expires_at` (set for Cloudflare-registered domains; null for external registrar) | works today |
| Registrar detail (auto-renew, lock, price) | `GET /accounts/{id}/reg/domains` | **blocked**: errors 7003 — needs Registrar Read |
| Emails sent per zone | `GET /zones/{id}/email/routing` (+ stats) | **blocked**: 10000 — needs Email Routing Read |
| Billing period certainty + usage-v2 | subscriptions + `/billing/usage-v2` | **blocked**: 10000 — needs Billing Read |
| Project consumption | `/api/billing` `projects[]` + `topConsumers[].project` | already collected — pure presentation |

One token edit (Billing Read + Email Routing Read + Registrar Read)
unblocks three features at once.

## Decisions

- **Domains wave 1 ships on zones alone** — no token dependency, cached at
  LIST_TTL (1h/24h). Registrar detail is wave 2 behind the token change.
- **Projects view adds zero upstream calls** — it re-shapes the existing
  billing payload. Cheapest high-value view in the backlog.
- **Sorting**: per-table top-right controls (sort-by + direction), numeric
  desc default (operator preference), persisted per table in localStorage;
  the WeakMap repaint-survival from tonight stays as the in-session layer.
- **Componentization** lands as one refactor wave AFTER the new views —
  extracting progress-bar/heading/filter-bar while three new views are
  being built would churn the same files twice. New views use the shared
  components from day one; old views migrate in the refactor wave.

## Filed

- FEAT-p9GD588 — domains view (wave 1 zones, wave 2 registrar)
- FEAT-p9B11X1 — email metrics (token-gated)
- FEAT-p0QKZT0 — projects view (zero new upstream calls)
- FEAT-pRC6EDA — table sort controls + saved preference
- IMP-pEY98JH — reusable components + motion/progress polish

Parent: ROAD-107 (Ops tier). Dispatch order once the GLM window reopens:
projects view → domains wave 1 → sort controls → componentization (email
metrics when the token allows).
