---
ulid: 01M4QX4BT11TXMQEYCNEVYMX43
title: 'Multi-user auth: connect your Cloudflare account + biometric app lock'
created: "2026-10-12T15:14:26.241534+04:00"
status: seed
source: human
origin:
    session: 76
---

# Multi-user auth: connect your Cloudflare account + biometric app lock

**Operator ask (2026-10-12):** when the public Cosmoflare app ships, can users "log in with Cloudflare" — an OAuth consent flow like Sign in with Google that grants the app scoped access to their account? And can the mobile app lock behind Face ID, beyond 2FA?

**Landscape notes (verify live before building — no public third-party OAuth was known as of 2026-10):**
- Cloudflare does NOT offer a public "Sign in with Cloudflare" OAuth/OIDC authorization flow for third-party apps to act on a visitor's account. Cloudflare Access authenticates users INTO apps you control (Google/GitHub/OTP identity), and can act as an OIDC provider for your own app — but it does not grant API access to the visitor's Cloudflare account.
- The standard multi-user pattern for CF tools: each user creates a scoped API Token (dash.cloudflare.com/profile/api-tokens, template deep-link pre-fills permission groups) and pastes it once. A polished onboarding flow (deep link to token creation with the exact scopes listed, then paste) is the shippable equivalent today.
- If CF ever ships a public OAuth/consent product, swap the token-paste onboarding for it behind the same session abstraction.
- Mobile biometric lock: fully feasible. React Native — expo-local-authentication / react-native-biometrics (Face ID/Touch ID/Android biometric + PIN fallback), and keep the CF token in the Keychain/Keystore with device-only, biometry-gated access so it never lives in app storage. PWA interim — WebAuthn passkeys give Face ID through Safari on iOS 16+ for login; full app-lock biometrics need the native app.

