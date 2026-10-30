# Control framework

The public HTTPS listener serves browser sessions, Admin and Member commands,
enrollment and provider webhooks. A separate TLS 1.3 listener verifies client
certificates for ordinary Agent traffic. `internal/auth.AgentAuthenticator` then
checks the current database identity and revocation state (§16.1). The binary
fails closed until the PKI task supplies this verifier; it never trusts
`X-Node-ID`, `X-Client-Cert` or forwarding headers.

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
and security headers → contract route/listener classification and per-peer ingress
limits → session or machine
authentication → role/MFA checks → per-principal rate and concurrency buckets →
Origin/CSRF and recent admin authentication → required key/If-Match headers →
class body cap (413 above it) → JSON well-formedness (400) → schema validation (422) → machine identity consistency
→ generated strict binding → command transaction/replay → domain handler.

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

Pre-authentication ingress buckets also bound credential/database work per
connection address and route class. IPv4 is one principal per address and IPv6
is one principal per /64; an unparseable connection address is refused with 429
(see [authentication](AUTH.md)). Rate buckets and concurrency gates are separate
for browser, agent control, heartbeat, lease and usage traffic; anonymous
enrollment and webhooks have another bucket on that same address key. The
loaded operation ids `EnrollNode` and `ReceiveProviderWebhook` use the
anonymous class, `SubmitObservation` uses heartbeat, `RenewServingLease` uses
lease, and `SubmitUsageBatch` uses usage. Body caps are separate from those
rate buckets. Browser writes use `browser_body_bytes` (default 1 MiB). Agent
control, heartbeat, lease renewal, and anonymous enrollment use
`agent_body_bytes` (default 1 MiB). Provider webhooks use
`webhook_body_bytes` (default 1 MiB). Usage batches use `usage_body_bytes`
(default 8 MiB). A body larger than its class cap is HTTP 413
`PAYLOAD_TOO_LARGE` (`Request body exceeds the configured limit.`) and the
extra byte is not parsed. A body within the cap then passes a JSON
well-formedness check, except webhook bodies, which stay raw for provider
signature verification. Up to `canonjson.MaxBytes` (1 MiB, the signed
canonical-document bound) the check is `canonjson.Canonicalize`. Above that
bound, and only through the class cap, one `json.Decoder` over a
`LimitReader` accepts a single JSON value and rejects trailing data. A
failure is HTTP 400 `MALFORMED_REQUEST` (`Invalid JSON document.`). Schema
validation follows (422 `VALIDATION_FAILED`). Machine-identity comparison
then reads the parsed body, including nested usage events, and returns 403
`INCARNATION_MISMATCH` when a node or incarnation field differs from the
certificate. `fvpn_control_rate_limited_total` counts HTTP 429 responses
only.

Auth routes are additionally bounded per connection address. Limits are
per process, with a bounded principal map; deployment replicas multiply capacity.
If horizontally scaling, budget limits across replicas or add a shared limiter.
A saturated limiter returns 429 with `Retry-After`, never acknowledges work.

## Configuration and startup

`fvpn-control -config control.json [-migrate-on-start]` reads strict JSON and then
`FVPN_<UPPERCASE_JSON_FIELD>` overrides. Lists and rate buckets use JSON values in
the environment; durations are integer seconds. The defaults and complete typed
setting list are in `internal/config/config.go`. `docs/control/config.example.json`
contains only deployment placeholders. Database DSN and optional OIDC client
secret are read from files; TLS key material is loaded from its file. Effective
configuration redacts file paths and OIDC settings and never includes contents.
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
(`limiter` in `{"address","subject"}`) for HTTP 429 responses (D-060). Startup
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
`max_rate_principals` bounds each instance independently; unauthenticated IP
rotation cannot consume or evict authenticated principal buckets. Existing
heartbeat/lease/usage classes retain independent per-principal keys. Every
framework 429 includes `Retry-After`, including `/auth/*` rate and concurrency
rejections. These remain process-local limits.
