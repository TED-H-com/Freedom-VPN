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
| `VITE_API_BASE_URL` | `/api/v1` | Base URL prefix for all 50 OpenAPI contract operations (revision 14) (`src/api/client.ts`). |
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

On hosts where `/tmp` is tmpfs, set `TMPDIR` for the entire target (including Chromium) to a unique, short disk-backed directory beneath `$HOME/agent-work/tmp/`, and remove it with Python after the run. Long worktree paths can exceed Chromium's Unix socket path limit. Do not use `/dev/shm` or another tmpfs for test data.

### Product-Driven Seeding (`apps/portal/e2e/dev-stack.mjs`, D-050 / D-057 / D-058)
`dev-stack.mjs` drives member groups, node enrollments, exit publication, and fleet update batches through the real control-plane and worker APIs:
1. **Member groups (`POST /api/v1/admin/groups`):** Creates `standard-exits` and `default-members` via the admin API.
2. **Two enrolled nodes (`POST /api/v1/admin/enrollments` + `POST /api/v1/agent/enroll`):** Enrolls `tw-tpe-01` (`amd64`, `nb-peer-node-1`, active reporter submitting `NodeCheck.detail`) and `jp-tyo-01` (`arm64`, `nb-peer-node-2`, secondary reporter that stays offline for the `waiting_online` update-batch target).
3. **Exit publication (`PUT /api/v1/admin/nodes/{id}/publication`):** Publishes `tw-tpe-01` with `group_ids` under `If-Match`, acknowledges the desired generation via `POST /api/v1/agent/observations`, and waits for `fvpn-worker` (`PublishExit` + `ReadExitState`) to project `exits.publication_observed = 'active'`.
4. **Serving leases (D-101, §16.3):** Each reporter requests `POST /api/v1/agent/lease-renewals` using its enrolled mTLS identity and fresh renewal/idempotency IDs. Observations copy central's lease ID and expiry exactly. The active node's heartbeat timer renews before expiry, even while an E2E test holds observations; teardown stops the timer. Grant-less leases retry after one heartbeat, doubling to at most 60 seconds. A matching, unexpired lease with no grants and nonempty desired bindings reports `awaiting_member_grants`, three failed `AWAITING_MEMBER_GRANTS` checks and one lease-stage error; it never claims forwarding is enabled. Empty desired bindings do not imply this error. The simulator still supplies test-only kernel, QoS, usage and updater evidence; this is control-plane E2E coverage, not real-node enforcement verification.

Seed steps log completion. A publication timeout includes the last SQL projection and the admin API's publication operation steps, node checks, recent errors and drift. The pending `agent_policy` / `data_path_probe` step carries its unchanged `error_code` beside the free-text `message`, for example `NODE_NOT_SERVICEABLE: member_forwarding_unconfirmed (lease_unconfirmed)`. Both the operation banner and Playwright timeout diagnostics preserve this text without interpreting it as an enum. A missing message never proves the gate passed: D-074 keeps the operation `running`, the gate `pending`, and continues polling for fresh evidence. See [publication-gate diagnostics](../nodes/2026-10-31-publication-gate-cause.md).

### D-093 harness audit — 2026-10-31, Follow-up 18

**13 of 15 audit entries remain open; D093-06 and D093-10 are retired.** The count tracks audit entries, including D093-05's unhandled receipt annotation; it is not a count of forged fields. Real central responses now supply serving lease IDs/expiry, applied-generation inputs and fetched DNS document metadata. Successful fixture swaps govern acknowledgement. Other local health, enforcement and updater claims remain simulated as listed below.

The harness trusts desired-state/lease transport responses without the production agent's signature verification, key-rotation handling or time-anomaly handling. DNS byte verification does not remove that limitation. This harness proves control-plane flows and projections, not real-node acceptance.

| Item | Status / harness-written value | Evidence and remaining limitation |
|---|---|---|
| D093-01 | **Open** — `forwarding_block_reasons`, `member_forwarding_enabled` | Lease-related reasons now derive from the issued lease and desired bindings. Other conditions remain unsimulated: publication intent, identity/freeze, policy validity, safety, metering/storage and release support. The updater fixture still supplies its own forwarding flag. A real agent combines central responses with local enforcement/health. Deferred broader decision-model parity. |
| D093-02 | **Open** — `checks`: `internet_only`, `meter_path`, `tunnel_probe` always passed, static details/timestamps | These require local kernel/counter/tunnel evidence, not just central responses. Deferred real agent/network lab coverage. |
| D093-03 | **Open** — `checks`: successful `qos` and `source_binding` remain simulated; `lease` and all three failure overrides now derive from actual lease state | A real agent verifies local rate classes and source admission after compiling central policy/grants. Passing these checks is still a simulator claim. Deferred beyond the required provisional lease shape. |
| D093-04 | **Open** — Observation `errors[]` defaults empty; lease failures now use `AWAITING_MEMBER_GRANTS`, `STALE_GENERATION`, `LEASE_EXPIRED`; successful updater result `error_code:null`; rollback fixture supplies its vector error | Real agent derives these from lease/policy/local apply and updater outcomes. Only the required lease vocabulary changed. Nonlease errors and updater outcomes remain simulated. |
| D093-05 | **Open** — `identity_frozen` | **Not hand-written anywhere in this harness.** It is a central receipt annotation; the observation sender currently discards the response after 200/202 and does not react to a true annotation. A real agent handles central identity rejection/freeze plus local identity state. Deferred receipt/fail-closed modeling. |
| D093-06 | **Retired (Follow-up 18)** — applied generations and binding revision | No `1` fallback or SQL-derived acknowledgement. The fixture stages a separate desired snapshot, fetches and validates any DNS document, then swaps it into applied state. Observations and lease renewals use that applied snapshot, preserving decimal strings exactly. Failure preserves the old snapshot and sends no new acknowledgement. This is an in-memory simulator apply, not durable kernel enforcement; updater outcomes remain D093-11. |
| D093-07 | **Open** — `current_release_id`/`last_good_release_id` begin at the seeded 1.0.0 ID, switch to central `intent.release_id` on simulated success, or back to the 1.0.0 fixture on rollback; `agent_version/agent_release` fixed 1.1.0 | Desired release IDs and manifest hashes come from central, but actual/last-good releases and versions require local verified installation/journal state. Deferred real updater execution. |
| D093-08 | **Open** — `provisioning_state:ready`, `safety_status:passed`, `data_health:healthy`; NetBird fixed peer/version/kernel backend; random boot/ledger/meter IDs | The peer is seeded into the fake provider; the other values are local runtime/provisioning measurements, not computed from central responses alone. Deferred real node health and durable identity integration. |
| D093-09 | **Open** — Usage `last_durable_event_seq:12`, ACK `10`, count `2`, bytes `1840`, age 30 seconds, archive `4096`, null blocked sequence | Real agent derives these from its durable meter/outbox and central ACKs. Fixed report-only fixture, not actual accounting evidence; deferred real metering/outbox feed. |
| D093-10 | **Retired (Follow-up 18)** — fabricated DNS ID/hash/generation and feed hash/count | Fetches the issued resolver document over enrolled agent mTLS and checks exact byte length and SHA-256 before the fixture swap. Evidence uses that document/reference and its actual feed hashes/counts (including an empty feed list); absent/null references emit null evidence. Bad fetches, hashes, sizes or feed counts abort the swap. The `applied` outcome denotes simulated application, not AdGuard read-back or real DNS enforcement. |
| D093-11 | **Open** — Update-result `succeeded` or rollback-vector state/stage/health/checkpoints/recovery flag; issued time plus 1/2 second timestamps | Intent IDs, generation and hash come from central; outcomes and timings require the updater journal/health checks. Deferred real updater reporting; current vectors remain control-plane projection tests. |
| D093-12 | **Open** — Direct SQL release/artifact records, `release_verifications.state=verified`; fleet desired/supported release IDs and `support_generation=1`; initial policy generations/revisions/activation | Release/policy content and fleet settings are fixture inputs. Verification state and manual activation/generation are shortcuts around product workflows, not agent-computed evidence. DNS activation already uses its API. Deferred API-driven release verification and remaining policy activation. API verification is blocked locally because the production fetcher rejects local addresses and ignores proxies ([`internal/releases/verify.go`](../../internal/releases/verify.go#L35)), the worker selects it unconditionally ([`apps/worker/jobs_nodes.go`](../../apps/worker/jobs_nodes.go#L333)), and node configuration has no fixture transport setting ([`internal/config/nodes.go`](../../internal/config/nodes.go#L16)); orchestrator routed this item to those owners. |
| D093-13 | **Open** — Direct SQL `usage_rollups` bytes/projection generation/gap/estimated flags | Projector-owned accounting output, not agent input; inherited large-integer UI fixture. Deferred ingestion of real usage events and waiting for the usage projector. |
| D093-14 | **Open** — Device restore fence + `device_authorization_observations.observed_authorized=false`; pending device row; node notes/resource-version bump | Inherited explicitly scoped restore/concurrency test fixtures; normal device discovery/revocation/re-grant still use product flows. Central owns these results; a real agent does not compute them. Deferred orchestration review of fixture exception scope. |
| D093-15 | **Open** — `devices.discover` outbox event and aggregate/schema version 1; billing customer seed; fake provider objects | Inputs used to drive real workers, not forged operation completion. A real agent does not own these central/provider records. Keep documented as test setup; review any future projector-boundary tightening. |

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

## Development dependency advisory — 2026-10-31, Follow-up 19

Dependabot alert #22 reports `minimatch < 11.0.1`,
[GHSA-g3hv-75wg-m249 / CVE-2026-118764](https://github.com/advisories/GHSA-g3hv-75wg-m249),
with `scope: development`. The alert remains open: the permitted dependency-only
repair breaks the current accessibility linter. Both `package.json` and
`package-lock.json` were restored byte-for-byte; no tool versions, lint rules,
portal behavior, or live-stack fixtures changed.

### Dependency paths and attempted repair

`npm ls minimatch --all` before the attempts and after restoration:

```text
@freedom-vpn/portal@0.1.0
├─┬ eslint-plugin-jsx-a11y@6.10.2
│ └── minimatch@3.1.5
├─┬ eslint@9.39.5
│ ├─┬ @eslint/config-array@0.21.2
│ │ └── minimatch@3.1.5 deduped
│ ├─┬ @eslint/eslintrc@3.3.7
│ │ └── minimatch@3.1.5 deduped
│ └── minimatch@3.1.5 deduped
└─┬ typescript-eslint@8.25.0
  └─┬ @typescript-eslint/parser@8.25.0
    └─┬ @typescript-eslint/typescript-estree@8.25.0
      └── minimatch@9.0.9
```

1. `npm audit fix` without `--force` exited 1, retained both versions, and still
   reported the advisory (12 moderate entries including dependent packages).
2. Adding `"overrides": { "minimatch": "^11.0.1" }` and running `npm install`
   resolved every path to 11.0.3. `npm audit --audit-level=moderate` reported zero
   vulnerabilities. A clean `npm ci` succeeded and typechecking passed, but
   `npm run lint` exited 2, including after deleting `node_modules` and reinstalling:

   ```text
   ESLint: 9.39.5
   TypeError: (0 , _minimatch.default) is not a function
   Rule: "jsx-a11y/label-has-associated-control"
   ```

   The failing input is the existing label at
   `apps/portal/src/pages/admin/AdminDnsPolicyPage.tsx:494`.
   `eslint-plugin-jsx-a11y/lib/util/mayContainChildComponent.js:38` calls the
   legacy callable export; minimatch 11 exposes named exports. A per-parent
   override that keeps this parent on 3.x cannot satisfy the requirement that
   every installed minimatch be at least 11.0.1. Disabling the rule or patching
   the dependent tool would exceed this dependency-only change.

### Reachability evidence

Minimatch is a glob matcher used here by development tooling only. Both locked
copies have `dev: true`; `npm ls minimatch --omit=dev --all` returns an empty
tree (exit 1). The dependency paths above accept lint configuration, file paths,
and accessibility component patterns, rather than portal request input.

The advisory describes amplification in `firstPhasePreProcess` with
`optimizationLevel >= 2`. The installed 3.1.5 implementation has no such pass.
The 9.0.9 copy defaults to level 1; its direct caller in
`@typescript-eslint/typescript-estree/dist/useProgramFromProjectService.js`
passes no options. That optional project service is not enabled in
`apps/portal/eslint.config.js`. These are evidence about the current callers,
not a claim that all uses of the affected versions are safe.

`npm run build -- --manifest --sourcemap` exited 0 with Vite 6.4.5:

```text
✓ 128 modules transformed.
dist/.vite/manifest.json          0.19 kB
dist/assets/index-BqlzUxff.css    7.76 kB
dist/assets/index-DsVXBaUd.js   538.42 kB
```

The generated manifest has one entry, `index.html`, referencing the JavaScript
and CSS above, with no additional imported chunks. Its JavaScript source map
lists 80 sources; none is minimatch, ESLint, or typescript-eslint. Together with
the empty production dependency tree for minimatch, this shows no path into the
built production portal bundle for this revision. The optional manifest/source maps
were generated for inspection only; build configuration is unchanged.

After restoration, `make -f mk/portal.mk portal-verify` passed the clean install,
typecheck, ESLint, formatting, 9 harness tests, 65 Vitest tests, coverage (85.1%
lines), and production build. `make -f mk/portal.mk portal-e2e` passed all 9
Playwright tests against freshly built Go control and worker binaries. The
isolated test database, processes, and disk scratch were removed afterward.

The repository owner should review the evidence and keep the alert open while
upstream dependents adopt a compatible patched minimatch API. Do not apply the
blanket override solely to clear the advisory. Any eventual dependency repair
must pass a clean install, `portal-verify`, and the live `portal-e2e` suite, plus
the PR's own Portal CI before merge. No alert dismissal was performed.
