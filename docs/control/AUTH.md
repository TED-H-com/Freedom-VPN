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
   The connection principal is the TCP peer (`r.RemoteAddr`) when
   `trusted_proxies` is empty, which is the default, and whenever that peer
   is outside the configured CIDRs. `X-Forwarded-For` is read only after the
   TCP peer matches a CIDR in `trusted_proxies`. The walk starts at the right,
   skips entries that are themselves inside a trusted CIDR, and stops at the
   first remaining address. At most 64 entries are examined. A missing header,
   or a window of only trusted hops, keeps the TCP peer. A later header line
   was appended by the next hop and is further to the right than an earlier
   line. Entries are trimmed. A forwarding entry and `RemoteAddr` share one
   parser. A bracketed address with no port (`[2001:db8::1]`) is a forwarding
   entry; the same text is not a connection address, because `RemoteAddr`
   carries a port. Bracketed input strips the brackets and any zone
   identifier, then calls `net.ParseIP`. It never splits a colon inside the
   brackets. `[2001:db8::1]:443` is host plus port. Unbracketed input with
   more than one colon calls `net.ParseIP` on the full string first. A
   trailing `:port` is removed only when that parse fails: eight hextets plus
   a port (`2001:db8:1:2:3:4:5:6:443`), or an IPv4-mapped address written with
   a port (`::ffff:192.0.2.77:44`). An unbracketed string that parses is that
   address, so `1::2:3:4:5:6` is `1:0:0:2:3:4:5:6` and is not the host
   `1::2:3:4:5`. Go's IPv6 `RemoteAddr` is `[ip]:port`, so the TCP peer still
   uses the bracketed form. Eight hextets written with a zero-width `::`
   (`2001:db8:1::2:3:4:5:6` for `2001:db8:1:2:3:4:5:6`) are that address.
   `net.ParseIP` rejects the zero-width marker; accepting it, before any port
   split, keeps the last hextet in the host. The `/64` stays
   `2001:db8:1:2::/64` and does not move into a trusted `2001:db8:1::/64`.
   The source port is not part of the key. `X-Real-IP` and RFC 7239 `Forwarded` are not
   read, including from a trusted peer. `Forwarded` has a structured grammar
   that is easy to mis-parse into a bucket the client chose; the proxies in
   this deployment append `X-Forwarded-For`. A trusted proxy that sends only
   `Forwarded` stays in the TCP peer's bucket.
   IPv4, including an IPv4-mapped IPv6 address, is one bucket per address.
   IPv6 is one bucket per /64 prefix, so one subscriber prefix cannot fill
   `max_rate_principals` (default 10000) with a distinct bucket per /128.
   There is no prefix-length setting. Bracketed (`[2001:db8:0:1::1]:443`) and
   unbracketed (`2001:db8:0:1::1:443`) forms share that /64 key: the unbracketed
   string parses as an address whose last hextet is inside the same /64.
   A zone identifier is not part of the key. Login routes (`/auth/login`,
   `/auth/callback`, and any other auth path, including `/auth/token`) use
   `"login:" + key`. `/auth/session` and `/auth/logout` use `"session:" + key`,
   a separate bucket with the same address rule.
   A TCP peer or a chosen forwarding entry that does not parse as IPv4 or IPv6
   fails closed with HTTP 429 `RATE_LIMITED` and `Retry-After` of at least one
   second. That request is not placed in a shared empty-host bucket and does
   not occupy a principal slot. An unparseable entry is not replaced by an
   entry further left. The same 429 increments
   `fvpn_control_rate_limited_total{class,limiter="address"}`.
   The anonymous class table and each class's ingress table on the public
   listener derive that same address key (IPv4 per address, IPv6 per /64).
   Each table is its own map, so the key inside a table is the connection
   principal. Login and session routes still share one map and distinguish
   `login:` and `session:` keys. The agent listener does not read
   `X-Forwarded-For`, even when its TCP peer is listed in `trusted_proxies`.
   A request with no verified client certificate is HTTP 401 and occupies no
   slot. A verified chain keys that class's ingress bucket before
   `AuthenticateAgent`, and takes the class concurrency slot before that
   lookup. The key is the node id in the leaf's `urn:fvpn:node:<uuid>` URI
   SAN, the same SAN `CA.Issue` writes, read from the certificate the TLS
   handshake already verified. Every still-usable serial of that node shares
   one burst, including certificates kept alive for `RenewalOverlapSeconds`.
   There is no database read on this path. A verified leaf with no canonical
   node URN keeps the SHA-256 of its DER (`r.TLS.VerifiedChains[0][0].Raw`).
   That fingerprint is only the separator for those leaves: it stops them
   collapsing into one bucket, and it does not join a real node's bucket.
   A failed authentication has already spent the token. The concurrency pool
   stays one pool per class of `max_concurrent`, which bounds how many of
   those lookups run together. A forwarding header does not choose either.
   mTLS subject keys stay the route class plus the node scope. The certificate
   authenticates the node, so a proxy in front of this listener does not move
   the ingress bucket.
   The login and session token buckets use this principal. The login
   concurrency gate stays one process-wide gate of `max_concurrent`.
   Production members commonly share a single source address behind
   carrier-grade NAT or enterprise gateways, and one OIDC sign-in takes two
   requests (`GET /auth/login` plus `GET /auth/callback`); a burst of 40 allows
   20 concurrent member sign-ins from one shared IPv4 address, or from one
   IPv6 /64, to complete while keeping session bootstrap and logout in a
   separate bucket. Behind a trusted proxy that address is the derived client
   principal, so each distinct client address has its own burst.
   A proxy that connects from several addresses is several CIDRs in
   `trusted_proxies` (each address as a /32, or one covering prefix). The
   shortest accepted prefix is /8 for IPv4 and /32 for IPv6. `0.0.0.0/0`,
   `::/0`, and any shorter prefix are rejected: trusting the world lets any
   peer choose its bucket. Write an IPv4 proxy as an IPv4 CIDR. An address
   that falls inside a trusted CIDR is a hop, so list proxy addresses only.
2. **Per-subject completion limiter (`login_subject_per_minute`, default `10` per minute):**
   Enforced in `Browser.callback` after OIDC token verification and keyed on the
   authenticated scoped subject `(issuer, client_id, sub)`. This bounds repeated
   login completions by a single account without starving other members who share
   the same NAT address.

Every HTTP 429 from either limiter includes `Retry-After` and the fixed
`RATE_LIMITED` error code (`apierr.RateLimited`), and increments
`fvpn_control_rate_limited_total{class,limiter}` on the control Prometheus
registry (`class="login"` or `"session"`, `limiter="address"` or `"subject"`).
The same counter records API 429 responses. An address or subject refusal uses
`limiter="address"` or `limiter="subject"` with the route class. A refusal from
the per-class concurrency gate uses `limiter="concurrency"` with that route
class. The counter does not move for a body-size 413 or a JSON 400.

API ingress is one table per class (`browser`, `agent`, `heartbeat`, `lease`,
`usage`, `anonymous`), each capped at `max_rate_principals` (default 10,000).
Enrollment and webhooks also use a separate anonymous class table with the same
cap. Authenticated subjects use another table, keyed by node or session. That
subject key does not use `X-Forwarded-For`. On the public listener the
pre-authentication address is the connection principal above, including a
client taken from `X-Forwarded-For` when the TCP peer is a configured trusted
proxy. On the agent listener the ingress key is the node URN on the verified
leaf, or the leaf fingerprint when that URN is absent, taken before
`AuthenticateAgent`. A request without a verified client certificate occupies
no slot. A failed authentication has spent the ingress token. IPv6 keying
stays /64. The anonymous ingress table and the anonymous class table also
keep a /48 group beside that /64 key. One /48 may hold every slot except a
reserve of one (`anonymousGroupReserve`, not a config key). At the default
cap of 10,000 that is 9,999 /64s. A cap of one still allows that single slot.
A new /64 in a /48 that already has keys is HTTP 429 once only the reserve
remains. It is not inserted and it does not evict another prefix. An existing
key in that /48 still spends a token. Another /48, or an IPv4 address, can
take the reserved slot. IPv4 is not grouped. The reserve is only on the
anonymous class, so a /48 sweep inserts nothing into the browser, agent,
heartbeat, lease, usage, or subject tables, and it cannot fill the anonymous
tables so tightly that an unrelated address has no slot. There is no separate
config key for the reserve. `max_rate_principals` (default 10,000) remains
the cap. At capacity a new principal is admitted by walking from the tail
and evicting the first bucket that is idle with a full token count, or has
been idle for more than one hour. The walk stops at that bucket. It continues
through a run of depleted tail buckets until it finds one or the table ends,
so eight depleted buckets cannot hide a full one. The long walk happens only
when the table is full and the tail is depleted, and it is bounded by
`max_rate_principals`. A table that finds no eligible bucket returns HTTP 429
with `Retry-After` of one second and inserts nothing. Ingress tables and the
anonymous class table each use one rate. The subject table mixes class rates.
The full-table walk is what lets a full browser bucket leave ahead of any
number of slower heartbeat tails.

## Browser class calibration

`classify` returns class `browser` and the `browser_rate` bucket for
administrator and member routes. Heartbeat stays `SubmitObservation`, lease
stays `RenewServingLease`, usage stays `SubmitUsageBatch`, anonymous stays
`EnrollNode` and `ReceiveProviderWebhook`, and an agent-tagged operation stays
agent. `browser_rate` in `internal/config.Defaults` is `per_second: 10`,
`burst: 30`. One session is one subject key (`browser:` plus the session
scope), so every browser route on that session shares the bucket. Pre-auth
ingress for this class uses `ingress_rate` (`per_second: 20`, `burst: 60`)
per connection principal. The subject budget is half that address budget on
both axes (`10/20` and `30/60`). Two operator sessions on one address can
each spend a full subject burst before the shared browser ingress table binds.

Portal mount fan-out on this bucket is one GET per call (`apps/portal/src`).
Node detail mounts `getNode` and `listAdminGroups` (2), and `getUpdateBatch`
when `node.update.batch_id` is set (3). That page does not call
`listNodeIncarnations`; the route is one GET per cursor. Audit mounts one
`listAuditEvents`. Each next page, previous page, or resource-type change is
one more GET. The search box filters the page already loaded. Member detail
mounts one `getMember`. Fleet updates mounts three at once (releases,
release-support policies, and the node list). When that list names a batch,
one `getUpdateBatch` follows and the node-list query stops, so the screen
issues four GETs in sequence and three at once. Overview mounts two,
plus the batch when a node carries `batch_id`. The node list, member list,
settings, DNS, fair-use, enrollment detail, and groups screens each mount
one. `GET /admin/devices` (the operator device list, including
`authorization_unconfirmed`) and
`POST /admin/members/{member_id}/devices/{device_id}/authorize` are the same
browser class: an admin session, with CSRF and `Idempotency-Key` on the command.
`GET /auth/session` uses the session login bucket, outside this class.
An in-flight operation polls at 60 ms, then 120 ms, 240 ms, and doubles up to
2 s. A node-detail write, its two invalidations, and those polls stay under
ten browser requests in the first second.

Burst 30 admits the heaviest mount (3) ten times over, and admits a composed
view of node detail with a batch, one incarnation page, and one audit page
(5 GETs) six times over, in one instant. Sustained paging is one request per
click against a 10/s refill. An empty burst returns to full in 3 seconds
(`30/10`). Admin list reads stay in this class, beside the six ingress tables
(`browser`, `agent`, `heartbeat`, `lease`, `usage`, `anonymous`). The per-node
agent ingress key, the mTLS order, and the anonymous `/48` reserve are unchanged.

## Request body limits

The API handler reads each body through `http.MaxBytesReader` at the class
cap from `classify`:

| Class | Routes | Setting | Default |
| --- | --- | --- | --- |
| browser | browser-tagged operations | `browser_body_bytes` | 1 MiB |
| agent | agent-tagged operations other than the rows below | `agent_body_bytes` | 1 MiB |
| heartbeat | `SubmitObservation` | `agent_body_bytes` | 1 MiB |
| lease | `RenewServingLease` | `agent_body_bytes` | 1 MiB |
| anonymous | `EnrollNode` | `agent_body_bytes` | 1 MiB |
| anonymous | `ReceiveProviderWebhook` | `webhook_body_bytes` | 1 MiB |
| usage | `SubmitUsageBatch` | `usage_body_bytes` | 8 MiB |

Checks run in this order:

1. **Size.** Above the class cap the response is HTTP 413 `PAYLOAD_TOO_LARGE`
   (`Request body exceeds the configured limit.`). The extra byte is not parsed.
2. **JSON well-formedness.** Webhook bodies skip this step; `httpapi.RawBody`
   keeps the capped bytes for the provider signature. Every other JSON body
   of at most `canonjson.MaxBytes` (1 MiB) must pass `canonjson.Canonicalize`.
   That 1 MiB bound is the signed-document limit and stays there. A larger
   body, which the usage cap allows through 8 MiB, is checked by a streaming
   walk in `internal/httpapi` with the same rules: valid UTF-8, no byte order
   mark, no unescaped control character (including NUL), no lone UTF-16
   surrogate in a `\u` escape, no duplicate key at any object level after
   unescaping, nesting at most `canonjson.MaxDepth` (256), and integers within
   ±(2^53−1) with no fraction, exponent, leading zero, or negative zero.
   Trailing data is rejected. The walk retains the nesting depth and the keys
   of objects that are still open. On an agent mTLS body the same walk reads
   every `node_id`, `node_incarnation_id`, and `incarnation_id`, including
   nested usage events. There is no second `json.Unmarshal` into an `any`
   tree for that comparison. A walk failure is HTTP 400 `MALFORMED_REQUEST`
   (`Invalid JSON document.`).
3. **Schema.** A body of at most 1 MiB, and a larger body whose schema uses a
   keyword the streaming checker does not cover, is validated by
   `openapi3filter.ValidateRequest`, which decodes the body into an `any`
   tree. A body above 1 MiB whose schema the streaming checker covers
   (object, array, string, integer, boolean, null, required properties,
   `additionalProperties`, item and length bounds, numeric bounds, `uuid`,
   `date-time`, and enum) is checked during the walk, and `ExcludeRequestBody`
   skips that tree. `SubmitUsageBatch` is in that set. A schema that uses
   `oneOf`, `anyOf`, `allOf`, `not`, a discriminator, `uniqueItems`,
   `multipleOf`, `patternProperties`, `if`, `prefixItems`, exclusive bounds,
   `const`, another format, or an unresolved reference stays on
   `openapi3filter` and does allocate the tree. Webhook bodies are excluded;
   the provider's JSON is not a contract schema. Schema failure is HTTP 422
   `VALIDATION_FAILED` (`Request does not match the API schema.`). A syntax
   failure stays HTTP 400 and is decided before the schema result. An
   operation with no owner returns HTTP 501 after these checks and does not
   decode the body again in the generated handler. An operation that has an
   owner still decodes into its generated struct inside the handler.
4. **Identity.** The fields the walk read must match the certificate. A
   non-string value in one of those fields is a mismatch. The comparison does
   not build a second document. A mismatch after a successful schema check is
   HTTP 403 `INCARNATION_MISMATCH`.

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
