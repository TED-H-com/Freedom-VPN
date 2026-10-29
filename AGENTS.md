# AGENTS.md: engineering contract

This file binds every contributor, human or AI agent. Read it completely before you change anything. An orchestrator assigns tasks, reviews every change and merges it; task prompts add task-specific scope on top of this file.

## 1. Authority and spec markers

- The product contract is [`docs/spec/PROJECT_SPEC.zh-TW.md`](docs/spec/PROJECT_SPEC.zh-TW.md) (Project Spec v1.0). Cite it as `§<chapter>.<section>`. Requirement IDs such as `ENR02`, `ISO01`, `MTR03`, `UPD05` come from §4. Acceptance cases `AT01` to `AT32` come from §19.
- Spec markers and how code treats them:
  - **必須** (must): mandatory for v1.
  - **建議預設** (recommended default): implement as a configurable parameter whose default is exactly the spec value.
  - **待決策** (pending business decision): never invent a production value. Ship a required setting or a clearly labeled test value (`TEST ONLY`).
  - **待實測** (needs measurement): code must not claim the capability is verified. Record what still needs real devices or networks.
- If the spec is ambiguous, implement the safer, fail-closed reading and list the question under "Spec questions" in your report. Never silently weaken a requirement.
- Out of scope for v1 (§1, §2.2 of the plan, `SCOPE01`): partner portals, logistics, source trust levels, maintenance scheduling, remote shell, endpoint MDM, TLS interception, own-brand apps. Do not add them.

## 2. Repository layout

| Path | Contents |
|---|---|
| `apps/control` | Central HTTP API (`fvpn-control`): auth, domain services, operations. External calls never run inside a request transaction. |
| `apps/worker` | Background worker (`fvpn-worker`): outbox, NetBird sync, billing reconciliation, usage projection, policy compilation. |
| `apps/portal` | Admin and Member web app (TypeScript, React, Vite). Holds no NetBird token or signing key. |
| `internal/...` | Shared central domain packages (db, store, netbird adapter, billing, dnspolicy, pki, ...). |
| `pkg/...` | Leaf packages shared by central and node code (`decstr`, `canonjson`, `envelope`, `apierr`, generated `api` types). |
| `node/agent` | Node agent (`fvpn-agent`): enrollment, desired state, enforcement, metering, heartbeat, leases. |
| `node/updater` | Node updater (`fvpn-updater`): download, verify, journaled switch, health check, rollback. |
| `node/enforce`, `node/internal/...` | Node-side libraries: nftables and tc rendering, local SQLite store, meter. |
| `contracts` | OpenAPI document, JSON Schemas, examples and golden vectors. |
| `db/migrations` | Forward SQL migrations (embedded). |
| `deploy` | Dev compose files, systemd units, deployment examples. |
| `packaging` | Installer, release build, manifest and signing tooling. |
| `tests` | Network lab, end-to-end and fault-injection suites, acceptance fixtures. |
| `docs` | Spec, ADRs, PoC evidence, operator and member guides. |
| `mk/*.mk` | Make targets per area. The root `Makefile` includes them. |

## 3. Technology baseline (decided)

- **Go 1.27**, single module `github.com/TED-H-com/Freedom-VPN` at the repo root, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`. Every Go binary must cross-compile for `linux/amd64` and `linux/arm64`.
- **Central services** are a modular monolith: `fvpn-control` and `fvpn-worker` share `internal/` packages. No microservices, no Kubernetes.
- **PostgreSQL 17** through `github.com/jackc/pgx/v5`. Migrations are SQL files in `db/migrations`, run by `github.com/pressly/goose/v3` from an embedded FS. No ORM.
- **Node targets:** Ubuntu 24.04 LTS on amd64 and arm64. Local durable state uses SQLite in WAL mode through `modernc.org/sqlite` (pure Go). The data plane uses kernel nftables (atomic `nft -f` transactions), `tc` HTB with fq_codel, the kernel WireGuard path and a pinned NetBird client.
- **Portal:** TypeScript (strict), React, Vite. One app with Admin and Member areas by role. UI language zh-TW by default, English second.
- **Contracts:** `contracts/openapi.yaml` (OpenAPI 3.0.3) is the single source of truth for the HTTP API. Go types are generated into `pkg/api`, TypeScript types into `contracts/ts`. Signed or hashed documents that are not HTTP bodies (release manifest, desired-state and lease envelopes) are JSON Schema 2020-12 files in `contracts/schemas`.
- **Canonical JSON:** RFC 8785 (JCS) for every hash or signature over JSON, implemented in `pkg/canonjson`. Numbers inside hashed or signed documents must be integers within ±(2^53−1).
- **Large integers:** bytes, sequence numbers, generations, revisions and resource versions are JSON decimal strings handled by `pkg/decstr` (§10.4, §14.1). They never pass through float64 or a JavaScript Number.
- **Signatures:** Ed25519 over the JCS bytes of the `signed` object, in the envelope `{"signed": {...}, "signatures": [{"key_id", "algorithm": "ed25519", "signature_b64"}]}` (`pkg/envelope`). Each purpose has its own keys: release, desired-state/policy, lease, recovery. Nodes hold public keys only.
- **Node identity:** ECDSA P-256 key and PKCS#10 CSR for mTLS. The private key is generated on the node and never leaves it.
- **IDs:** UUIDs from `github.com/google/uuid` (prefer v7 for new rows). External IDs (NetBird, OIDC, payment provider) are opaque strings.
- **HTTP and logs:** standard library `net/http` routing and `log/slog` with redaction. No global mutable state.
- **Dependency freshness:** when you add or upgrade a module, pin the latest stable release (`go list -m -versions <module>` or `go get <module>@latest`, then `go mod tidy`), never whatever the local module cache happens to hold. Known-vulnerable versions are not acceptable on `main`; check `govulncheck ./...` (`go run golang.org/x/vuln/cmd/govulncheck@latest`) before you finish if you touched `go.mod`.
- **Pre-approved dependencies:** pgx/v5, goose/v3, google/uuid, modernc.org/sqlite, golang.org/x/{net,sync,crypto,sys,time}, coreos/go-oidc/v3, golang.org/x/oauth2, oapi-codegen (runtime and generator), getkin/kin-openapi (tests), santhosh-tekuri/jsonschema/v6, google/go-cmp, gopkg.in/yaml.v3, prometheus/client_golang. Any other dependency needs a justification in your report: license, maintenance, why the standard library is not enough.

## 4. Cross-cutting contracts

- API prefix `/api/v1`. JSON is UTF-8. Times are UTC RFC 3339 with `Z`. Intervals are half-open `[start, end)`.
- Decimal strings match `^(0|[1-9][0-9]*)$`, have at most 20 digits and are at most 18446744073709551615.
- Error envelope: `{"error": {"code", "message", "request_id", "retryable", "details"}}` with codes from §14.6 and §13.11 (`pkg/apierr`). Never return stack traces, SQL, certificates or secrets.
- Side-effecting POSTs accept `Idempotency-Key` (§14.5). Mutable resources use `ETag` and `If-Match` (412 on mismatch, 428 when missing). Asynchronous effects return 202 with an operation. External effects count as done only after read-back evidence.
- Desired state and observed state are always separate fields. A 202 is never shown or reported as completed.
- **Fail closed:** a missing or expired policy, lease, meter path or isolation check turns member forwarding off and keeps the control channel up.
- **Secrets:** never log tokens, private keys, Authorization headers or webhook secrets. Secret-bearing responses carry `Cache-Control: no-store`. Enrollment tokens are stored only as hashes.
- **Privacy (SEC02):** no full DNS query names, URLs, destination lists or packet contents in logs, metrics or the database.
- **Node paths** (defaults, configurable, tests use temp dirs): `/opt/vpn-node/releases/<release_id>`, `/opt/vpn-node/current`, `/var/lib/vpn-node/{identity,state,update-journal}`, `/etc/vpn-node/`.
- **Node network ownership:** only the nftables table `inet fvpn` and objects we create. Interfaces and namespaces we create start with `fv`. Never `flush ruleset`, never touch other tables, never rewrite the host default route or host DNS.

## 5. Testing

- `go test ./...` passes with no network, no root and no Docker.
- Heavier tests use build tags and skip with a clear message when their prerequisite is absent:
  - `integration`: needs `FVPN_TEST_DATABASE_URL`, an admin DSN. Each test creates and drops its own database through `internal/db/dbtest`.
  - `netlab`: the privileged Docker network lab in `tests/netlab`.
  - `netbird_integration`: the dev NetBird deployment in `deploy/dev/netbird`.
  - `adguard_integration`: the dev AdGuard Home in `deploy/dev/adguard`.
- Name tests after the spec ID they prove, for example `TestAT15_ContiguousAck` or `TestMTR03_DuplicateSameHash`.
- Golden files live in `testdata/` next to the test, or in `contracts/vectors/` when several languages share them.
- Frontend checks run inside `apps/portal`: `npm ci`, `npm run typecheck`, `npm run lint`, `npm test`, `npm run build`.
- Each area defines its targets in `mk/<area>.mk`. Every target name starts with `<area>-` (for example `contracts-verify`), so included files never collide. Each file works alone (`make -f mk/<area>.mk <area>-test`) and when the root `Makefile` includes `mk/*.mk`. To join the root aggregates, append to `BUILD_TARGETS`, `TEST_TARGETS`, `LINT_TARGETS` or `VERIFY_TARGETS` with `+=`. Recipes run from the repository root and must not assume any other area's file is present.

## 6. Multi-agent protocol

- Each task runs in its own git worktree on branch `<task-id>`, created from `main`. Work only in that worktree.
- Touch only the paths your task owns, as listed in your prompt. You may also change `go.mod` and `go.sum` (through Go tooling), your own `mk/<area>.mk` and your own `.github/workflows/<area>.yml`. Do not edit `AGENTS.md`, `README.md`, `docs/spec/**`, `.gitignore` or another area's files. If you need such a change, ask under "Requests to orchestrator" in your report.
- Commit early and often on your branch. Message format: `<area>: <summary>`, with the trailer `Agent: <cli>/<model>`. Never push, merge, tag, or rewrite `main`.
- Self-check before you finish:
  1. Build, vet and all relevant tests pass. Paste the real output tails into the report.
  2. Formatting and lint are clean: `gofmt -l` empty, eslint clean, and the root `make go-lint` (gofmt, vet, staticcheck pinned in `mk/go.mk`) exits 0 for the whole module. Your `<area>-lint` target runs the same pinned staticcheck over your own packages.
  3. `git status` is clean and everything is committed.
  4. `git diff --stat main...HEAD` lists only owned paths and the allowed shared files.
  5. The hygiene check in §8 passes.
- Write the hand-off report to `.agent/REPORT.md` in the worktree root. It is gitignored and never committed. Sections: Summary; Requirement and AT coverage; Files; How to run; Self-check results; Decisions made; Spec questions; Known gaps and later work; Requests to orchestrator.
- Then stop. The orchestrator reviews, may send a follow-up run on the same branch, and merges.

## 7. Safety on the shared development host

- Never use `sudo`. Never change host networking: no `ip`, `nft`, `iptables`, `tc`, `sysctl` or `modprobe` on the host. Network experiments run only inside the privileged lab container from `tests/netlab`.
- Docker: only create containers, images, networks and volumes named `fvpn-<task-id>-*`, and remove them when you finish. Never stop, remove, exec into or reconfigure any other container. The shared test PostgreSQL behind `FVPN_TEST_DATABASE_URL` must never be dropped or reconfigured; create and drop only your own databases.
- Never generate traffic toward third-party hosts. Load and throughput tests run only inside the lab. Outbound network access is for dependencies, container images and public documentation.
- Do not read files outside your worktree except toolchains, module and package caches, and public documentation. Never read SSH keys, credential stores, other repositories or home-directory configuration.

## 8. Public repository hygiene

- This repository is public. Never commit secrets, real or realistic-looking. Never commit personal names or emails, private hostnames, absolute home paths, addresses of real infrastructure or account identifiers.
- Examples use `example.com` or `example.invalid`, IPv4 documentation ranges `192.0.2.0/24`, `198.51.100.0/24` and `203.0.113.0/24`, IPv6 `2001:db8::/32`, and overlay addresses in `100.64.0.0/10`.
- Test keys are generated at test time, or derived from a seed committed under `testdata/` and labeled `TEST ONLY - NOT A SECRET`.
- Hygiene check: `git diff main...HEAD | grep -nE '/home/|BEGIN (RSA|EC|OPENSSH) PRIVATE KEY|sk_live_|ghp_|gho_'` must find nothing outside clearly labeled test fixtures.

## 9. Definition of done (Project Plan §6.1)

- The deliverable runs, maps to requirement IDs, and tests the normal case plus at least one important failure case. A UI alone or a successful API call alone is not done.
- Work with external state proves that retries do not duplicate resources, and the UI distinguishes desired from applied state.
- Work with private keys, enrollment codes or payment signatures proves that logs, responses and the frontend do not leak them.
- Forwarding, metering and update work is verified on amd64 and arm64. At minimum, arm64 cross-compiles in CI; real arm64 runs are recorded as evidence when hardware is available.
- Docs, configuration examples and migrations change together with the code. No important manual step lives only in someone's memory.
