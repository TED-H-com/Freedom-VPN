# Control framework

The public HTTPS listener serves browser sessions, Admin and Member commands,
enrollment and provider webhooks. A separate TLS 1.3 listener verifies client
certificates for ordinary Agent traffic. `internal/auth.AgentAuthenticator` then
checks the current database identity and revocation state (§16.1). The binary
fails closed until the PKI task supplies this verifier; it never trusts
`X-Node-ID`, `X-Client-Cert` or forwarding headers as identity. Rate-limit
connection principals on the public listener follow `trusted_proxies` below.
The agent listener does not read them.

`contracts/openapi.yaml` remains the source of all API operations. oapi-codegen
v2.8.1 generates the standard-library strict server and embedded specification.
Generated aliases reuse `pkg/api`, and a small local generator creates partial
handler composition. `make -f mk/control.mk control-drift` checks all three outputs.
See the [generator documentation](https://github.com/oapi-codegen/oapi-codegen/blob/main/README.md)
for its strict-server model. Runtime validation also uses the existing
kin-openapi dependency because generated binding alone does not enforce all
required fields, enums, limits or `additionalProperties: false`.

The core implements Admin/Member operations and Admin audit reads. Domains add
their handlers through composition. Unregistered operations return HTTP 501
NOT_IMPLEMENTED with retryable=false after security and input checks (revision 3).
This transitional response must be gone from documented endpoints in a release.
Domain handlers must not construct error response objects: return `*apierr.Error`
for safe errors, or an ordinary error for a generic 500.
Report-only codes (ARTIFACT_INVALID, ISOLATION_BASELINE_MISSING and
METER_EPOCH_CHANGE) returned as handler errors become 500 INTERNAL_ERROR. The
framework logs the known code and central request ID; it never logs the error
message or details.

## Request boundary

Order: central UUID v7 and isolated client trace context → buffered panic recovery
and security headers → contract route/listener classification. On the agent
listener a request without a verified client certificate is HTTP 401 and is
inserted into no limiter table. A verified chain then takes the class ingress
bucket, keyed by the SHA-256 of the leaf certificate, and the class concurrency
slot, both before `AuthenticateAgent`. A failed authentication has spent that
token. On the public listener the order is the connection principal, then the
class ingress bucket, then session authentication, then the subject or anonymous
bucket, then the class concurrency slot. Both listeners then apply role/MFA
checks, Origin/CSRF and recent admin authentication, required key/If-Match
headers, the class body cap (413 above it), JSON well-formedness (400), schema
validation (422), machine identity consistency, and, for an operation that has
an owner, generated strict binding, the command transaction, and the domain
handler. An operation with no owner returns HTTP 501 after those checks and
does not decode the body again.

Access logs contain only central request ID, normalized route template, status
and duration. They exclude every request/response header, URL/query, body, panic
value and internal error text. Client tracing is available separately through
`httpapi.ClientTrace`; domains must never log it as trusted metadata. Errors use
`pkg/apierr.Write`. The response buffer discards partial output on panic. Responses
are bounded to 32 MiB; APIs here have no streaming endpoints. All API and auth
responses use `Cache-Control: no-store`, except `GET /api/v1/agent/resolver-documents/{document_id}`
with HTTP 200 which sets `Cache-Control: private, max-age=0` so agents can
revalidate cached resolver documents with `ETag` / `If-None-Match` (D-056); 304
and error responses on that route still use `Cache-Control: no-store`. HSTS,
nosniff, frame denial, restrictive CSP and no-referrer headers apply. Portal
static assets need their own web server.

Browser writes require exact configured Origin plus a session-bound CSRF token.
All administrator writes require recent authentication; this intentionally applies
the high-impact rule conservatively (§16.1). Unknown fields in allowlisted write
schemas fail before domain code. Semantic domain constraints listed in
`docs/contracts/README.md` remain each owning service's responsibility.

Pre-authentication ingress is one table per route class (`browser`, `agent`,
`heartbeat`, `lease`, `usage`, `anonymous`). Each table holds
`max_rate_principals` connection principals (default 10,000). IPv4 is one
principal per address and IPv6 is one principal per /64 (D-060). The anonymous
ingress table and the anonymous class table also keep a /48 budget beside that
/64 key: one /48 may hold at most half of `max_rate_principals` buckets (5,000
at the default cap, and at least one when the cap is one). A new /64 past that
budget is refused with 429, is not inserted, and does not evict another prefix.
IPv4 is not grouped. The budget is only on the anonymous class, so the sweep
inserts nothing into the browser, agent, heartbeat, lease, usage, or subject
tables. Enrollment and webhooks take their class slot in the separate anonymous
table under the same /64 key and the same /48 budget. The authenticated subject
table is a further instance, keyed by node or session, with the same cap and
no /48 budget. An unparseable connection address, including an unparseable
`X-Forwarded-For` entry chosen from a trusted peer, is refused with 429 and
occupies no slot (see [authentication](AUTH.md)). An mTLS request without a
verified client certificate is HTTP 401 and occupies no slot in any table. A
verified chain keys the agent listener's ingress bucket by the SHA-256 of the
leaf certificate before `AuthenticateAgent`, and a failed authentication has
spent that token.

`trusted_proxies` (default empty) is the list of CIDRs whose TCP peers may
supply `X-Forwarded-For`. An empty list keeps every public-listener
pre-authentication bucket on the TCP peer, so a forwarding header cannot
choose a bucket. The agent listener's ingress key is the certificate
fingerprint either way. Production
either terminates TLS on `fvpn-control` or sets `trusted_proxies` to the
proxy's source addresses. A proxy that connects from several addresses is
several CIDRs in that list: each source address as a /32, or one covering
prefix that is still at least /8 for IPv4 or /32 for IPv6. `0.0.0.0/0`,
`::/0`, and any shorter prefix are rejected, because trusting the world lets
any peer choose its bucket. Write an IPv4 proxy as an IPv4 CIDR. When the TCP
peer is inside a listed CIDR, the public listener walks `X-Forwarded-For`
from the right, skips trusted hops, and stops after 64 entries. The chosen
address is the connection principal for the per-class ingress tables, the
anonymous table, and the browser login and session token buckets (IPv4 per
address, IPv6 per /64). Parsing keeps a bracketed host whole, parses an
unbracketed multi-colon string before stripping a trailing port, and accepts
eight hextets written with a zero-width `::` as that address, so a trailing
hextet cannot move a /64 into a trusted prefix. An address that falls inside
a trusted CIDR is itself a hop, so the list is proxy addresses only. The
agent listener does not read the header. Its ingress key is the certificate
fingerprint, so a proxy in front of it does not choose the bucket. Subject
buckets stay on the node or session scope. The login
concurrency gate remains one process-wide gate of `max_concurrent`. Access
logs record the request ID, route, status, and duration. Prometheus
labels are route, status, class, and limiter. The peer address is absent
from both.
`X-Real-IP` and RFC 7239 `Forwarded` are not read. Replicas do not share
limiter state, so each replica applies its own budget and the deployment
budget is the per-process budget times the replica count.

When a table is full, a new principal is admitted by scanning eight
least-recently-used buckets from the tail and evicting the first that is idle
with a full token count or has been idle for more than one hour. The window
is constant, so the scan stays O(1). One depleted tail cannot hide a newer
bucket inside that window that has already refilled. More than eight depleted
buckets at the tail can still refuse a newcomer while a full bucket waits
further toward the head, and that newcomer receives HTTP 429 with
`Retry-After` of one second. Together with the anonymous /48 budget, that is
the bound on a full table refusing newcomers. Ingress tables and the anonymous
class table each use one rate. The subject table mixes class rates. The
eight-bucket window is what lets a full browser bucket leave ahead of a slower
heartbeat tail.

Rate buckets and concurrency gates stay separate for browser, agent control,
heartbeat, lease, and usage traffic. The loaded operation ids `EnrollNode`
and `ReceiveProviderWebhook` use the anonymous class, `SubmitObservation`
uses heartbeat, `RenewServingLease` uses lease, and `SubmitUsageBatch` uses
usage. Body caps are separate from those rate buckets. Browser writes use
`browser_body_bytes` (default 1 MiB). Agent control, heartbeat, lease
renewal, and anonymous enrollment use `agent_body_bytes` (default 1 MiB).
Provider webhooks use `webhook_body_bytes` (default 1 MiB). Usage batches use
`usage_body_bytes` (default 8 MiB). A body larger than its class cap is HTTP
413 `PAYLOAD_TOO_LARGE` (`Request body exceeds the configured limit.`) and the
extra byte is not parsed. A body within the cap then passes a JSON
well-formedness check, except webhook bodies, which stay raw for provider
signature verification. Up to `canonjson.MaxBytes` (1 MiB, the signed
canonical-document bound) the check is `canonjson.Canonicalize`. Above that
bound, through the class cap, a streaming walk in `internal/httpapi` applies
the same rules: valid UTF-8, no byte order mark, no unescaped control
character (including NUL), no lone UTF-16 surrogate in a `\u` escape, no
duplicate key at any object level after unescaping, nesting at most
`canonjson.MaxDepth` (256), and integers within ±(2^53−1) with no fraction,
exponent, leading zero, or negative zero. Trailing data is rejected. The walk
retains the nesting depth and the keys of objects that are still open. On an
agent mTLS body it also reads every `node_id`, `node_incarnation_id`, and
`incarnation_id`, including nested usage events, so identity does not
`json.Unmarshal` the body into a second `any` tree. Failure is HTTP 400
`MALFORMED_REQUEST` (`Invalid JSON document.`). A body of at most 1 MiB, and
a larger body whose schema the streaming checker does not cover, then goes
through `openapi3filter.ValidateRequest`, which decodes an `any` tree. A body
above 1 MiB with a covered schema, including `SubmitUsageBatch`, is checked
during the walk and `ExcludeRequestBody` skips that tree. Schema failure is
HTTP 422 `VALIDATION_FAILED`. A node or incarnation field that differs from
the certificate is HTTP 403 `INCARNATION_MISMATCH`, after the schema check.
`fvpn_control_rate_limited_total` counts HTTP 429 responses only.

Auth routes are additionally bounded per connection principal (the TCP peer,
or the client derived from `X-Forwarded-For` when that peer is listed in
`trusted_proxies`). Limits are per process, with a bounded principal map;
deployment replicas multiply capacity. If horizontally scaling, budget limits
across replicas or add a shared limiter. A saturated limiter returns 429 with
`Retry-After`, never acknowledges work.

## Configuration and startup

`fvpn-control -config control.json [-migrate-on-start]` reads strict JSON and then
`FVPN_<UPPERCASE_JSON_FIELD>` overrides. Lists and rate buckets use JSON values in
the environment; durations are integer seconds. `trusted_proxies` defaults to
an empty list. `FVPN_TRUSTED_PROXIES` is a JSON array of CIDRs, for example
`["203.0.113.0/24","2001:db8:1::/48"]`. The defaults and complete typed
setting list are in `internal/config/config.go`. `docs/control/config.example.json`
contains only deployment placeholders. Database DSN and optional OIDC client
secret are read from files; TLS key material is loaded from its file. Effective
configuration redacts file paths and OIDC settings and never includes contents.
`trusted_proxies` is not a secret and is left visible.
Do not put DSNs or secret contents directly in configuration/environment values.
The example includes all five domain files, in central config order:
`members_config_file`, `dns_config_file`, `nodes_config_file`,
`policies_config_file`, and `usage_config_file`. Supply the corresponding domain
settings as described in [composition](COMPOSITION.md#lifecycle-and-responsibilities).

Use a DDL-capable connection only for explicit migrations; use a least-privileged
runtime identity otherwise. Apply embedded migrations before either process starts.
The five-digit migration `00100_control.sql` adds session/login-flow tables,
operation metadata, delivery evidence and dead-letter state. Periodic control
cleanup deletes expired login/session/idempotency scratch only; it never removes
member, operation, billing or audit history. Audit retention remains the database
retention policy (recommended 180 days); service-log rotation should enforce the
recommended seven days (§16.4).

Public TLS and agent mTLS use the configured certificate/key. The agent CA is
separate configured trust. `/healthz` indicates the process can respond; `/readyz`
pings PostgreSQL. Agent health probes also need a valid client certificate. Signals
stop acceptance and drain requests under a bounded shutdown timeout. Read-header,
read, write and keepalive deadlines are startup settings. Startup logs intentionally
omit underlying provider/driver/TLS error strings, which can contain credentials.

The control metrics listener serves plain HTTP `GET /metrics` on `metrics_address`
(default `127.0.0.1:9091`, loopback IP literals only), using the registry shared
with domain builds. It exports `fvpn_control_http_requests_total{route,status}`
with bounded normalized route labels and `fvpn_control_rate_limited_total{class,limiter}`
(`limiter` in `{"address","subject","concurrency"}`) for HTTP 429 responses (D-060).
The per-class API concurrency gate uses `limiter="concurrency"`. Login and session
concurrency refusals on `/auth/*` use `limiter="address"`. Startup
logs record the actual bound address, including an ephemeral port when configured
as zero. Metrics starts and drains with the other listeners; a listener failure
stops the application. Control and worker need distinct metrics ports when hosted
together.

No latency/SLA, fleet enforcement timing, real device or arm64 runtime claims are
made. Both architectures are cross-compiled; §16.5 targets require a separate
recorded load/device evaluation.

## Checks

```sh
make -f mk/control.mk control-verify
make -f mk/control.mk control-test-integration
make go-lint
```

Integration tests use `internal/db/dbtest`, require `FVPN_TEST_DATABASE_URL`, and
create/drop only random isolated databases. Make recipes use disk scratch under
`.agent/tmp` and remove it on exit. Unit tests use local httptest providers and
need no external network, root or Docker.

## Toolchain security

Control CI uses Go 1.27.3. The supplied Go 1.27.1 baseline has reachable standard
library advisories; Go 1.27.3 fixes the reported issues. See the official
[Go release history](https://go.dev/doc/devel/release#go1.27.3). Deployment and the
repository-wide Go workflow should use at least this patched release; the shared
minimum directive remains unchanged for the orchestrator to coordinate.

Ingress and authenticated traffic use separate bounded limiter instances (§14.6).
Each class has its own ingress table, enrollment and webhooks have a separate
anonymous class table, and authenticated subjects have another. `max_rate_principals`
bounds each instance independently. One IPv6 /48 can hold at most half of
`max_rate_principals` /64s in each anonymous table (5,000 at the default cap)
and cannot evict a browser, agent, heartbeat, lease, usage, or subject bucket.
A full table scans eight tail buckets before it refuses a newcomer.
Existing heartbeat/lease/usage classes retain independent per-principal keys.
Every framework 429 includes `Retry-After`, including `/auth/*` rate and
concurrency rejections and the API concurrency gate. These remain process-local
limits.
