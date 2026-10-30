# Portal Local Development & Control Service Proxy (`DEV.md`)

This guide explains how to run `apps/portal` in three modes:
1. **Live Control Plane Stack (`portal-dev-stack` & `portal-e2e`):** Runs real `fvpn-control` and `fvpn-worker` against an isolated PostgreSQL database from `$FVPN_TEST_DATABASE_URL` with test OIDC and fake NetBird/Stripe servers.
2. **Live Control Service Mode (`fvpn-control` proxy):** Proxies `/api/v1` and `/auth` from the Vite dev server to a local `fvpn-control` process.
3. **Contract-Mocked Mode (`VITE_API_MOCK=1` in dev & Vitest):** Runs standalone in the browser/Vitest using MSW handlers backed by `contracts/examples/valid/*.json`. Never included in production builds (`import.meta.env.PROD`).

---

## Environment Variables

All Vite and E2E environment variables are optional and have contract-compliant defaults:

| Variable | Default | Description |
|---|---|---|
| `VITE_API_BASE_URL` | `/api/v1` | Base URL prefix for all 44 OpenAPI contract endpoints (`src/api/client.ts`). |
| `VITE_AUTH_BASE_URL` | `/auth` | Base URL prefix for browser session routes (`GET /auth/session`, `/auth/login`, `POST /auth/logout` in `src/api/auth.ts`). |
| `VITE_API_MOCK` | `0` in prod build (`VITE_API_MOCK=1` only in dev) | Set to `1` in non-production dev to enable browser MSW mocking; production builds strip MSW completely. |
| `VITE_API_PROXY_TARGET` | `http://127.0.0.1:8080` | Target origin for Vite's dev proxy when `VITE_API_MOCK=0`. |
| `PORTAL_E2E_BASE_URL` | `http://127.0.0.1:4173` | Target URL for Playwright end-to-end tests (`apps/portal/playwright.config.ts`). |

---

## Mode 0: Full Live Control Plane Stack (`portal-dev-stack` & `portal-e2e`)

Requires `$FVPN_TEST_DATABASE_URL` (an admin PostgreSQL 17 DSN). Creates an isolated `fvpn_w3_portal_live_*` database, runs migrations, starts `fvpn-control` and `fvpn-worker` with test OIDC, fake NetBird, and fake Stripe servers, enrolls a live mTLS node, drives an initial fleet update batch and mTLS update result (`POST /api/v1/admin/update-batches` → `GET /api/v1/agent/desired-state` → `POST /api/v1/agent/update-results` → `POST /api/v1/agent/observations`), and serves the portal on `http://127.0.0.1:4173` (dropping the isolated database on exit):

```bash
# Start interactive live stack on http://127.0.0.1:4173
make -f mk/portal.mk portal-dev-stack

# Run Playwright E2E suite against the live stack
make -f mk/portal.mk portal-e2e
```

### Remaining D-050 Fixture Deviations (`apps/portal/e2e/dev-stack.mjs`)
Until the dev stack runs a second fake agent reporter and route publication projector, only two node/exit fixture seeds remain in `dev-stack.mjs`:
1. **`exits` publication state seed:** Sets `publication_intent = 'published'` and `publication_observed = 'active'` on the enrolled node's `exits` row so member exit listings and update serving checks see an active published exit.
2. **`SEEDED_NODE_2_ID` offline fixture node:** Inserts a secondary offline node (`nodes`, `node_incarnations`, `exits` for `jp-tyo-01` / `arm64`) without an agent reporter so `POST /api/v1/admin/update-batches` snapshots a second target in `waiting_online` (`unconfirmed: true`, `policy_generation: null`, `policy_refs: null`).

---

## Mode 1: Running Standalone with Contract Mocks (Default)

```bash
cd apps/portal
npm ci
npm run dev
```

- Opens on `http://localhost:5173`.
- MSW intercepts `/auth/session`, `POST /auth/logout`, and `/api/v1/*` using fixtures imported from `contracts/examples/valid/*.json`.
- A top-bar **Mock Persona Switcher (TEST ONLY)** allows switching between `admin`, `admin_no_mfa`, `anonymous`, and all member entitlement states (`member_active`, `member_throttled`, `member_pending_payment`, `member_cancel_at_period_end`, `member_grace`, `member_expired`, `member_suspended`), as well as simulating `401 SESSION_EXPIRED`.

---

## Mode 2: Running Against a Local `fvpn-control` Service

When running `fvpn-control` locally (default listen address `http://127.0.0.1:8080`):

```bash
cd apps/portal
VITE_API_MOCK=0 VITE_API_PROXY_TARGET=http://127.0.0.1:8080 npm run dev
```

### How the Vite Dev Proxy Works (`apps/portal/vite.config.ts`)
When `VITE_API_MOCK=0`, Vite disables the browser Service Worker and configures its built-in HTTP proxy on `http://localhost:5173`:
- `/api/v1/*` → `http://127.0.0.1:8080/api/v1/*` (`changeOrigin: true`)
- `/auth/*` → `http://127.0.0.1:8080/auth/*` (`changeOrigin: true`)

Because both `/api/v1` and `/auth` are served from the same browser origin (`http://localhost:5173`), `credentials: 'include'` cookies (`HttpOnly` session cookie and readable `__Host-fvpn-csrf` cookie) and `X-CSRF-Token` headers work identically to production without cross-origin cookie issues.

### Custom Base URLs (Without Proxy)
If `fvpn-control` is mounted behind a reverse proxy or custom prefix, you can also configure the client prefixes directly:

```bash
VITE_API_MOCK=0 \
VITE_API_BASE_URL=/api/v1 \
VITE_AUTH_BASE_URL=/auth \
npm run build
```
