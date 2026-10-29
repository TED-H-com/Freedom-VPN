# Portal Authentication, Session Bootstrap & CSRF Flow

This document describes the browser session lifecycle, role/MFA guards, login redirect sanitization, logout, and CSRF double-submit protection implemented in `apps/portal` (§14.1, §16.1, §18, AT28).

---

## 1. Boundary Between `/auth/*` and `/api/v1/*`

- Browser session routes live under `/auth` (configurable via `VITE_AUTH_BASE_URL`, default `/auth`), outside `/api/v1`, and are not part of the 41-endpoint OpenAPI contract.
- Browser role selection (`?role=admin|member`) is supported on `/auth/session`, `/auth/login`, and `/auth/logout` per `x-fvpn-browser-auth` (D-032, D-042).
- All `/auth/*` client helpers and types are isolated in a single small module: [`apps/portal/src/api/auth.ts`](../../apps/portal/src/api/auth.ts).

| Route | Method | Purpose |
|---|---|---|
| `/auth/session` | `GET` | Bootstraps current browser authentication state (`Cache-Control: no-store`). |
| `/auth/login?return_to=<relative path>` | `GET` (Full-page redirect) | Starts the OIDC authorization code + PKCE flow on `fvpn-control`. |
| `/auth/callback` | `GET` | Handled by `fvpn-control` after OIDC provider callback; sets session & CSRF cookies and redirects to `return_to`. |
| `/auth/logout` | `POST` | Invalidates the server-side session and clears session cookies. Requires `X-CSRF-Token`. |

---

## 2. Session Bootstrap (`GET /auth/session`)

On startup, `SessionProvider` (`src/context/SessionContext.tsx`) calls `GET /auth/session` with `credentials: 'include'`.

The server responds with HTTP `200 OK` and `Cache-Control: no-store`:

### Anonymous (no active session)
```json
{
  "authenticated": false
}
```

### Authenticated Member or Admin
```json
{
  "authenticated": true,
  "role": "admin",
  "display_label": "ops-admin@example.com",
  "mfa_verified": true
}
```
For members, `member_id` (UUID) is also included when linked.

### UI Guard Behavior (`RoleGuard.tsx`)
1. **`anonymous` (`authenticated: false`):** Renders the sign-in prompt (`data-testid="auth-anonymous-prompt"`) with a full-page link to `/auth/login?return_to=<current relative path>`.
2. **`member` (`role: "member"`):** Allowed on `/member/*`. Blocked on `/admin/*` with HTTP 403 Forbidden view (`data-testid="role-guard-forbidden"`).
3. **`admin` without MFA (`role: "admin"`, `mfa_verified: false`):** Blocked on `/admin/*` (`data-testid="role-guard-mfa-required"`, §16.1) with a prompt to complete MFA verification via `/auth/login?return_to=<current relative path>`.
4. **`admin` with MFA (`role: "admin"`, `mfa_verified: true`):** Allowed on `/admin/*`.

Note: Client-side route guards are defense-in-depth UX only; `fvpn-control` independently enforces session validity, role, and MFA on every `/api/v1/admin/*` and `/api/v1/member/*` request (AT28).

---

## 3. Login Redirect & Open-Redirect Prevention

- `buildLoginUrl(returnTo)` and `sanitizeReturnTo(returnTo)` in `src/api/auth.ts` enforce that `return_to` is strictly a relative path starting with a single `/`.
- Absolute URLs (`https://...`, `http://...`), protocol-relative URLs (`//evil.example.com`), backslash paths (`/\\...`), and non-path schemes (`javascript:`, `data:`) are rejected and normalized to `'/'`.

---

## 4. Logout (`POST /auth/logout`)

- Clicking the Sign Out button (`data-testid="logout-btn"`) sends `POST /auth/logout` with `credentials: 'include'` and the `X-CSRF-Token` header.
- Afterward, `SessionContext` invalidates cached queries and transitions the UI to the unauthenticated state.

---

## 5. CSRF Double-Submit Protection (`__Host-fvpn-csrf` → `X-CSRF-Token`)

`fvpn-control` uses the double-submit cookie pattern for browser state-changing requests (§16.1):

1. The server sets a readable (non-`HttpOnly`, `Secure`, `Path=/`, `SameSite=Lax`) cookie named `__Host-fvpn-csrf`.
2. At request time, `readCsrfToken()` in `src/api/auth.ts` reads `document.cookie` and extracts `__Host-fvpn-csrf`.
3. Every unsafe browser request (`POST`, `PATCH`, `PUT`, `DELETE`, including `POST /auth/logout`) attaches the cookie value in the `X-CSRF-Token` request header.
4. If the `__Host-fvpn-csrf` cookie is absent, the client still sends the request without `X-CSRF-Token` and lets the server decide (`403 CSRF_INVALID` if required).
5. Safe `GET` requests (`GET /auth/session`, `GET /api/v1/*`) **never** include `X-CSRF-Token`.
