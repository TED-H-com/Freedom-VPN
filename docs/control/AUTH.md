# Browser and machine authentication

These browser routes live outside the `/api/v1` contract:

| Route | Behavior |
|---|---|
| `GET /auth/login` | Creates a short-lived, browser-bound PostgreSQL flow (recording the requested `role=member\|admin`, default `member`); redirects to the configured issuer using Authorization Code, S256 PKCE, state, nonce and `max_age=0`. |
| `GET /auth/callback` | Consumes the matching one-use state/browser binding, exchanges the code, verifies ID-token signature/issuer/audience/`azp`/expiry/nonce, enforces the per-subject login completion limit, resolves identity for the flow's requested role and rotates the session. Redirects to `/`. |
| `POST /auth/logout` | Requires the current session, exact Origin and `X-CSRF-Token`; deletes the server session and expires its cookie. Returns 204. |
| `GET /auth/session` | Returns 200 with `{"authenticated":false}` for absent/expired sessions. Otherwise returns authenticated, role, display_label, mfa_verified and member_id only for members. Returns the CSRF token in `X-CSRF-Token`. |

Every route has `Cache-Control: no-store`. The portal should bootstrap at
`/auth/session` and use its CSRF response header on writes. The portal's former
`/api/v1/session` path is intentionally not an API alias.

`__Host-fvpn-admin`, `__Host-fvpn-member` and `__Host-fvpn-login` cookies are opaque, Secure, HttpOnly,
SameSite=Lax, Path=/, and have no Domain attribute. Only session/state/browser
hashes persist as lookup credentials. A transient PKCE verifier and nonce are
stored server-side for the short login TTL and removed when consumed or expired.
No ID/access/refresh token is retained. Sessions enforce idle and absolute limits,
with absolute expiry additionally capped by the verified ID-token expiry.

Admin and member session cookies coexist. API paths under `/api/v1/admin/` and
`/api/v1/member/` select only their matching cookie; a role query never overrides
that selection. `/auth/session?role=admin|member` and `/auth/logout?role=admin|member`
select the browser role explicitly when both cookies are present. With zero or
one cookie the selector is ignored (revision 3); a sole cookie is selected
automatically. Two cookies without a valid selector return 400. The selected
session supplies the CSRF token. Logout deletes and expires only that session.
OIDC callback chooses the role from verified IdP claims and rotates only its role's
cookie. The old `__Host-fvpn-session` cookie is no longer accepted; sign in again.
There remains one short-lived login flow cookie per browser.

Identity is canonical `(issuer, subject)` via `internal/members/issuer.Canonical`
(D-052 F-04 / F-09). `client_scope` is recorded
as diagnostic metadata (last OIDC client scope seen) and is never part of identity:
an existing `identity_bindings` row with a different `client_scope` is reused, its
`client_scope` and `verified_at` are updated in place in the same transaction
(migration `00186`), and `identity_binding.proof_updated`
(`after: {"identity_binding_id","reason":"proof_updated"}`)
is appended once for the rotation on the D-029 allowlist without raw subject or
issuer fields. Email is
neither requested nor used for linking. A new identity, `pending_payment` member
and `member.register` audit event are created atomically only when no `(issuer, subject)`
binding exists; a canonical `(issuer, subject)` advisory lock plus a savepoint-backed
insert-or-select retry on unique violation (`identity_bindings_subject_key` or
`identity_bindings_issuer_subject_key`) prevents concurrent first logins from creating
duplicate members across schema versions. Administrators use `admin_identities` and
`admin_sessions` (migration 00101), resolved and updated by canonical `(issuer, subject)`
under the same rule; admin logins never insert `members` or `identity_bindings`.
Admin and member identities remain distinct even for the same `(issuer, subject)`.
`GET /auth/login` accepts an optional `?role=member|admin` query parameter (or
infers `admin` from `return_to=/admin...`, defaulting to `member`) and binds the
requested role in `oidc_login_flows.role` (migration `00185_oidc_login_flows_role.sql`;
`ConsumeFlow` continues to accept the legacy `role:verifier` prefix for one release)
so a dual-role account can sign in as a member
and create its `members` and `identity_bindings` rows (D-052, F-11). Requesting
`admin` requires the configured ID-token admin claim (returning 403 `FORBIDDEN`
when absent), `auth_time`, and configured AMR/ACR MFA evidence (returning 403
when MFA is missing). Recent authentication is enforced again for admin writes.
`/auth/callback` also validates the OIDC `azp` (Authorized Party) claim per OIDC
Core §3.1.3.7 rules 4–5 / RFC 9068 (F-05): whenever `azp` is present or `aud`
contains more than one audience, `azp` must equal `oidc_client_id`; otherwise the
callback returns 401 `UNAUTHENTICATED` and records a token-free `session.login`
denial audit (`reason: "azp_mismatch"`). Closed accounts cannot log in or use
existing member sessions. Further account linking is out of scope.

Every successful `PostgresStore.Login` (session creation or rotation) appends
`session.login` in the same transaction. When a login on a shared browser
replaces an existing session belonging to a different principal (`oldID != newID`),
`PostgresStore.Login` also appends `session.logout` with `after: {"reason":"superseded"}`
for the superseded principal in the same transaction (F-11). Logout appends
`session.logout` with its session deletion. Members use `member:<member UUID>`.
Administrators use `admin_identity:<admin principal UUID>` (revision 3, D-037),
referencing the administrator's auth-store identity. This rule also applies to
administrator identity subjects in session-revocation or role-denial audits;
never infer the subject's identity or role from a cookie name supplied by the
browser. The actor is the role plus its UUID; no subject, token or display label
enters the audit diff. The auth package emits only these closed-set, enum-conformant
session actions through `db.AppendAudit` because the audit HTTP reader imports auth.

## Login burst and per-subject rate limiters (D-060)

Browser authentication applies two complementary token-bucket limiters that can
be raised in a lab via `internal/config` (`docs/control/config.example.json`)
and can never be disabled (startup validation rejects zero burst or zero refill):

1. **Per-source-address limiter (`login_rate`, default `burst: 40`, `per_second: 1` / 60 per minute):**
   Enforced in `httpapi.BrowserRoutesWithMetrics` before handler execution.
   The key is only the connection address (`r.RemoteAddr`). Forwarding headers
   (`X-Forwarded-For`, `X-Real-IP`, `Forwarded`) and the source port are ignored.
   IPv4, including an IPv4-mapped IPv6 address, is one bucket per address.
   IPv6 is one bucket per /64 prefix, so one subscriber prefix cannot fill
   `max_rate_principals` (default 10000) with a distinct bucket per /128.
   There is no prefix-length setting. Bracketed (`[2001:db8:0:1::1]:443`) and
   unbracketed (`2001:db8:0:1::1:443`) forms of the same address share that key.
   A zone identifier is not part of the key. Login routes (`/auth/login`,
   `/auth/callback`, and any other auth path, including `/auth/token`) use
   `"login:" + key`. `/auth/session` and `/auth/logout` use `"session:" + key`,
   a separate bucket with the same address rule.
   A connection address that does not parse as IPv4 or IPv6 fails closed with
   HTTP 429 `RATE_LIMITED` and `Retry-After` of at least one second. That
   request is not placed in a shared empty-host bucket and does not occupy a
   principal slot. The same 429 increments
   `fvpn_control_rate_limited_total{class,limiter="address"}`.
   The anonymous enrollment and webhook address burst, and the
   pre-authentication ingress bucket, derive their address key the same way
   (`anonymous:` and `ingress:` prefixes in `internal/httpapi`).
   Production members commonly share a single source address behind
   carrier-grade NAT or enterprise gateways, and one OIDC sign-in takes two
   requests (`GET /auth/login` plus `GET /auth/callback`); a burst of 40 allows
   20 concurrent member sign-ins from one shared IPv4 address, or from one
   IPv6 /64, to complete while keeping session bootstrap and logout in a
   separate bucket.
2. **Per-subject completion limiter (`login_subject_per_minute`, default `10` per minute):**
   Enforced in `Browser.callback` after OIDC token verification and keyed on the
   authenticated scoped subject `(issuer, client_id, sub)`. This bounds repeated
   login completions by a single account without starving other members who share
   the same NAT address.

Every HTTP 429 from either limiter includes `Retry-After` and the fixed
`RATE_LIMITED` error code (`apierr.RateLimited`), and increments
`fvpn_control_rate_limited_total{class,limiter}` on the control Prometheus
registry (`class="login"` or `"session"`, `limiter="address"` or `"subject"`).

The production config requires HTTPS issuer/origin. Tests use a local signed-token
httptest provider with PKCE validation and negative signature, issuer, audience,
expiry, state, nonce, MFA and future-auth-time cases.

Agent routes require direct mutual TLS and a current database check through:

```go
type AgentAuthenticator interface {
    AuthenticateAgent(context.Context, *x509.Certificate) (auth.Principal, error)
}
```

The verifier must return Role=Agent, NodeID and IncarnationID, reject revoked or
superseded identities, and distinguish forwarding-disabled nodes from permanently
revoked control identities. Return it as `control.Built.Agents` from the PKI domain build in the
[composition seam](COMPOSITION.md). The framework compares every body node/incarnation field to
this identity, including nested usage events. A mismatch is 403
INCARNATION_MISMATCH. Unverified peer certificates and spoofed headers never
reach the verifier. `AgentAuthenticatorFunc` is available for tests.

Enrollment is the sole public Agent operation. The enrollment domain owns token,
CSR and proof verification; its idempotency scope includes a hash of the credential.
Provider webhooks are also public, and billing owns signature/replay verification.
`httpapi.RawBody(ctx)` returns the original bounded bytes before JSON binding; use
those exact bytes for provider verification. Never acknowledge until durable inbox
commit. Unregistered enrollment and webhook handlers return 501.
