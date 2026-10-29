# Control and worker composition

The binaries own configuration loading, the shared PostgreSQL pool and process
signals. Domains are built once with those dependencies before listeners or jobs
start. Adding a domain requires exactly one constructor line in each applicable
list: `control.Domains()` in `apps/control/handlers.go` and `worker.Domains()` in
`apps/worker/jobs.go`. All domain-specific construction belongs in
`wire_<area>.go` or `jobs_<area>.go`. No per-domain calls belong in either `main`.

## Control API

```go
type Dependencies struct {
    Config config.Config
    Pool *pgxpool.Pool
    Browser *auth.Browser
    Logger *slog.Logger
}
type Domain struct {
    Name string
    Build func(context.Context, Dependencies) (Built, error)
}
type Built struct {
    Handlers []any
    Agents auth.AgentAuthenticator
    Middleware func(http.Handler) http.Handler
    Runners []func(context.Context) error
}
func Domains() []Domain
func Servers(context.Context, Dependencies, ...Domain) (*Application, error)
func Run(context.Context, *Application) error
```

`Servers(ctx, dependencies)` builds the production list. Explicit variadic domains
replace that list for composition tests; they do not append to it. A build must
return only its own strict handler sets, not an already composed handler. The
shared composition layer calls `httpapi.Compose` once, rejecting overlapping
operation IDs, and creates the public and agent API handlers with `httpapi.New`.
Duplicate domain names, nil runners, and multiple non-nil `Agents` providers are
startup errors. Zero authenticators leaves agent authorization fail closed. A
single authenticator is used for both listeners; direct mTLS verification and
listener-specific route restrictions are unchanged.

Request flow is `httpapi.Transport → route selection → domain middleware in
registration order → strict API authentication/validation/command handling`.
For registrations A then B, A runs before B on entry and after B on return.
Middleware receives the tracked response and request ID, applies on both API
listeners, and must preserve the request context and original body. It does not
wrap browser auth routes, health or readiness probes. The shared transport keeps
panic recovery, safe error responses and access logging outermost.

`Application.Public` and `Application.Agent` expose the constructed listeners for
host/testing integration. Domains must never replace their `Handler`. The
control-core domain supplies operations/audit handlers and a supervised periodic
authentication/idempotency cleanup runner. Its transient cleanup errors retain
the existing safe warning/retry behavior.

## Worker API

```go
type Dependencies struct {
    Config config.Config
    Pool *pgxpool.Pool
    Metrics *prometheus.Registry
    Logger *slog.Logger
}
type Domain struct {
    Name string
    Build func(context.Context, Dependencies) (Built, error)
}
type Built struct {
    Jobs Registry
    Context func(context.Context) context.Context
    Runners []func(context.Context) error
}
func Domains() []Domain
func Open(context.Context, Dependencies, ...Domain) (*Application, error)
func (*Application) Run(context.Context) error
func (*Application) Once(context.Context) (bool, error)
```

`Open` builds the production list unless explicitly replaced for tests. It merges
registries and rejects duplicate event names before constructing the sole
fenced dispatcher. Context decorators add immutable values in registration order
and preserve cancellation/deadlines. All decorators run before any dispatcher or
runner starts, so one domain can publish a typed dependency consumed by another
without relying on build order. They must not start work, replace the parent
context, or acquire resources. `Once` uses the same decorators for deterministic
dispatch tests. A nil metrics registry gets a new process-local registry.

The application owns the dispatcher, runners and existing metrics/health
listener. There is no second outbox claimant hidden in a domain. Unknown events
still retry and dead-letter; successful work still requires a persisted receipt.
`/healthz` remains liveness and `/readyz` remains a bounded database ping.

## Lifecycle and responsibilities

Both processes log successfully built domain names and their sorted operation or
event names, never secrets or configuration values. A failed setting can return
`config.SettingError`; the binary logs only its setting name, not the wrapped
error or file contents. Members uses this for `members_config_file`.

Builds may validate explicit settings, read configured secret files and construct
clients/stores. They must not read flags or environment variables, open their own
pool, call `httpapi.New`, wrap `http.Server.Handler`, launch background goroutines,
or perform transactional external effects. Return background work as runners.
Builds should not acquire resources that need a separate shutdown lifecycle.

A listener, dispatcher or runner returning stops the application. A runner error
is returned as a process failure, cancels siblings and starts graceful shutdown.
SIGTERM/SIGINT also cancel the process context. HTTP drain and runner joins share
one `ShutdownSeconds` deadline; a timeout returns an error and closes listeners.
Runners must honor cancellation and wait for any children they start before
returning. There is no way to forcibly kill a Go goroutine; timed-out binaries
exit, leaving unfinished leased work recoverable. No success is fabricated.

## Members worked example

`members(),` is the only members registration line in each list. Control's build
loads the required `Dependencies.Config.MembersConfigFile`, constructs immutable
services using the supplied pool, returns `httpadapter.Handlers` with those
services, and returns `httpadapter.RequestContext` as middleware. The middleware
retains just the original bounded `Stripe-Signature` header; the framework still
provides original body bytes through `httpapi.RawBody`. There is no
`ConfigureMembers` or post-construction transport wrapper.

The worker build loads the same required settings and constructs the same domain
services. It returns `MemberJobs()`, a context decorator calling `WithMembers`,
and the `RunMemberSchedule` runner. This activates billing inbox/session dispatch,
reconciliation and device discovery in the actual worker binary. No business
value is inferred: missing, invalid or unreadable `members_config_file` fails
startup, as do unavailable required secret files. See
[member configuration](../members/README.md#configuration).

The node/policy domain can return a context decorator using
`membersruntime.WithCompiler(ctx, configuredCompiler)`. Member/device jobs read
that compiler when they run, after all decorators have applied. Until a
production compiler is supplied, policy-dependent jobs fail/retry without
claiming node enforcement. No TEST ONLY compiler, plan rate or signer is a
production fallback. The node team still owns persisted node acknowledgements.

## Migration for in-flight branches

To `w3-wire-nodes`, `w3-wire-policies-usage` and `w3-dns-publish`:

- Replace `wireNodeServers`, any second `httpapi.New`, path-switch wrapper and
  post-construction `server.Handler` assignment with a `Domain.Build` result.
  Return strict handler sets in `Built.Handlers`, the node/PKI authenticator in
  `Built.Agents`, and `httpwire.RequestContext` (or its equivalent) in
  `Built.Middleware`. Only one domain may provide `Agents`.
- Replace old `Handlers(pool, log)` or `Jobs()` registration edits with one
  `yourDomain(),` line in each `Domains()` list. Put all dependencies and settings
  in the domain files, using `Dependencies.Config` and the shared pool. Do not
  change `main`, add a second dispatcher, or wrap another domain's handler.
- Return worker registries, context decorators and runners through `Built`.
  Publish the production compiler through `membersruntime.WithCompiler`; member
  jobs consume it at execution time. Keep desired state and observed ACKs separate.
- Share the [NetBird binding and lock convention](../members/README.md#netbird-binding-and-lock-namespace-convention)
  for all writers. Do not independently invent a binding encoding or lock space.

Composition tests can substitute domain builds to supply fake external clients;
a separate subprocess integration test starts the real control/worker binaries
with explicit settings, exercises TLS login, member routes, signed Stripe
webhook replay and durable scheduled work, and verifies clean SIGTERM shutdown.
