# Browser and machine authentication

These browser routes live outside the `/api/v1` contract:

| Route | Behavior |
|---|---|
| `GET /auth/login` | Creates a short-lived, browser-bound PostgreSQL flow; redirects to the configured issuer using Authorization Code, S256 PKCE, state, nonce and `max_age=0`. |
| `GET /auth/callback` | Consumes the matching one-use state/browser binding, exchanges the code, verifies ID-token signature/issuer/audience/expiry/nonce, resolves identity and rotates the session. Redirects to `/`. |
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

Identity is `(issuer, client scope, subject)`. Email is neither requested nor used
for linking. A new identity, pending_payment member and audit event are created
atomically; a scoped advisory lock prevents concurrent first logins from creating
duplicate members. Administrators use `admin_identities` and `admin_sessions`
(migration 00101); admin logins never insert `members` or `identity_bindings`.
Admin and member identities remain distinct even for the same scoped subject.
The session role comes solely from the configured ID-token claim. A matching string or string
array claim grants the Admin role only when configured AMR/ACR MFA evidence is
present. Missing MFA returns 403. Admin auth_time is required; recent authentication
is enforced again for writes. Closed accounts cannot log in or use existing
member sessions. Further account linking is out of scope.

Every successful `PostgresStore.Login` (session creation or rotation) appends
`session.login` in the same transaction. Logout appends `session.logout` with its
session deletion. Members use `member:<member UUID>`. Administrators use
`admin_identity:<admin principal UUID>` (revision 3, D-037), referencing the
administrator's auth-store identity. This rule also applies to administrator
identity subjects in session-revocation or role-denial audits; never infer the
subject's identity or role from a cookie name supplied by the browser.
The actor is the role plus its UUID; no subject, token or display label enters the
audit diff. The auth package emits only these closed-set, enum-conformant session
actions through `db.AppendAudit` because the audit HTTP reader imports auth.

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
