# Portal Local Development & Control Service Proxy (`DEV.md`)

This guide explains how to run `apps/portal` in two modes:
1. **Contract-Mocked Mode (default in dev & tests):** Runs standalone in the browser/Vitest using MSW handlers backed by `contracts/examples/valid/*.json`.
2. **Live Control Service Mode (`fvpn-control` proxy):** Proxies `/api/v1` and `/auth` from the Vite dev server to a local `fvpn-control` process.

---

## Environment Variables

All Vite environment variables are optional and have contract-compliant defaults:

| Variable | Default | Description |
|---|---|---|
| `VITE_API_BASE_URL` | `/api/v1` | Base URL prefix for all 40 OpenAPI contract endpoints (`src/api/client.ts`). |
| `VITE_AUTH_BASE_URL` | `/auth` | Base URL prefix for browser session routes (`GET /auth/session`, `/auth/login`, `POST /auth/logout` in `src/api/auth.ts`). |
| `VITE_API_MOCK` | `1` in dev, `0` in prod build | Set to `1` to enable browser MSW mocking; set to `0` to disable MSW and use the Vite dev proxy / real backend. |
| `VITE_API_PROXY_TARGET` | `http://127.0.0.1:8080` | Target origin for Vite's dev proxy when `VITE_API_MOCK=0`. |

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
