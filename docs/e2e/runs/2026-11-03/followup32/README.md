# Follow-up 32 — isolation F19-02 publisher conflicts

2026-11-03. The client fix, synthetic cases, owned-site audit and ordinary live
serving proof are delivered. **The requested real-API 412→successful retry proof
is not exercised.** Its single planned invocation failed before enrollment due
to a local admission-wrapper error. It was retained and not retried, honoring
the prompt's no-run-retry instruction. F19-02 is not fully validated yet.

The fix follows spec §14.5 / API01: three writes at most, ten seconds total per
command, exact-code conflict re-reads, intended-value checks, unchanged UUID
key/body, and complete ordered receipts. Capacity values, publication request,
step order and downstream completion predicates are preserved. The
[audit](audit.md) lists every owned and read-only AT site and traces why 412 is
not stored under its key. The original isolation 19 run remains unchanged; an
observation race is plausible but its competing write was not identified.

## Every live invocation

Both stacks were fresh (`E2E_RESET=1`), sequential, prefix `e2e`, port base
`18000`, on source `fff680c6c2cd539b77cc534ce9d090f8ad9a8ec3`.

| Invocation | UTC bounds | Result |
|---|---|---|
| [Forced switch on](e22872b7-1112-4082-8e2b-235275f4a659/run-result.json), `e22872b7-1112-4082-8e2b-235275f4a659` | 23:06:10.847403Z–23:06:41.422688Z | go test exit 1; setup failed before enrollment or publisher; zero conditional writes, zero real 412s. |
| [Ordinary switch unset](c52b760c-c289-4220-a722-6c5a059819b9/acceptance-summary.md), `c52b760c-c289-4220-a722-6c5a059819b9` | 23:10:07.717977Z–23:10:59.511374Z | go test exit 0; 8 pass / 0 blocked / 0 fail; two 202 writes, zero conflicts. |

The first invocation's private admission wrapper printed `free` into Docker
stdout. `lab.stage` unmarshals that output as one JSON document, so the
pre-enrollment `waiting_token` predicate could not succeed. The original test
log and completed admin-login record are retained. Its acceptance summary
contains only that completed scenario; the companion `run-result` records the
actual failing invocation. A diagnostic read at 23:07:56.240904Z through the
corrected wrapper returned `{"stage":"waiting_token"}` from the same node.
The wrapper now captures admission output separately before executing a tool.
This was harness execution plumbing, not an API or agent failure. No source
change or third live invocation was used to replace the failed attempt.

The ordinary [publisher record](c52b760c-c289-4220-a722-6c5a059819b9/publish.json)
retains PATCH `If-Match="7"` → 202 and PUT `If-Match="8"` → 202, with UTC
bounds and each UUID key. Operation `04ec055f-e59c-4d28-9ee3-d17a8fd1a70c`
succeeded and `publication_observed=active`. The
[command receipt](c52b760c-c289-4220-a722-6c5a059819b9/publisher-retry-proof.json)
read back one durable accepted operation row per key. Its pass is the ordinary
command-key check; `test_only_stale_etag=false` and `real_conflicts=0` explicitly
mean it is **not live conflict-retry evidence**.

The independent [serving record](c52b760c-c289-4220-a722-6c5a059819b9/publisher-serving.json)
joins accepted report `01a1a3ad-e32f-7bd2-be5d-9dda91cbfb77`, populated lease
`a0cd9e18-aeb6-4b98-ab75-42bf76c78787`, one eligible member and generation `5`.
It checks matching desired/applied/local/kernel generations, forwarding and a
fresh serving heartbeat. Native member admission also counted 1113 upload and
1519 download bytes. These are fixture observations, not platform or timing
acceptance claims.

## Synthetic checks and gates

[All invocations and real output tails](checks.md) retain two initial lint
failures, their fixes, the later transport-change rechecks and the live failure.
The [final synthetic output](synthetic-transport.txt) covers PATCH, PUT and
revoke with one conflict then success, all-conflict exhaustion, already-intended
re-read, every tested other status, failed reads/transports, exact decimal
versions, the actual ten-second bound, and no implicit replay or redirect.
Three attempts remain blocked under persistent contention. The unchanged
original HTTP call is shared with the stub tests.

Required e2e/isolation lint, non-stack e2e tests, handover tests and root Go lint
passed on the final implementation. Both linux/amd64 and linux/arm64 builds
passed; arm64 is compile-only. Final evidence hygiene, formatting, diff checks
and owned inventory are recorded in `checks.md`. No dependency, product code,
AT-owned file, expected-blocked entry or Go embed was changed.

## Reproduction

Set TMPDIR and GOTMPDIR to `<worktree>/.agent/tmp`, use `GOCACHE="$(go env GOCACHE)"`,
`CGO_ENABLED=0`, `GOTOOLCHAIN=local`, and `PYTHONDONTWRITEBYTECODE=1`. Before every
make/go test/npm/docker invocation, including nested tools, run
`bash "$FVPN_HOLD_SH"` and start only on `free`; otherwise check every 60 seconds
in a bounded foreground loop. Capture successful admission output separately
from a tool's stdout/stderr so parsers receive the actual command output.
No reservation or shell cleanup trap is needed.

The invocation is:

```sh
FVPN_E2E_FOLLOWUP32=1 go test -v -tags e2e -count=1 -timeout 18m ./tests/e2e -run '^TestE2E_Followup32Publisher$'
```

Prepare each fresh stack with admitted `make -f mk/e2e.mk e2e-up E2E_RESET=1`;
tear it down with admitted `make -f mk/e2e.mk e2e-down`. The forced invocation
additionally sets `FVPN_E2E_PUBLISH_STALE_ETAG=1`; ordinary invocation unsets it.
The switch sends `"0"` only on the first PATCH and first PUT, never edits agent
reports or stored product state, and is off by default.

### Requests to orchestrator

2026-11-03:

1. Authorize a later fresh-stack forced-conflict proof after reviewing the
   preserved pre-enrollment failure. This session does not retry that run.
   Keep the failed ID and do not count the ordinary zero-conflict receipt as
   satisfying the real-412 requirement.
2. Route the [AT-owned conditional-write audit](audit.md#requests-to-orchestrator)
   to isolation, secops, enrollment, members, usage and updates. All their paths
   remained read-only; preserve expected-error tests.
3. Isolation owns its post-merge acceptance run and all 12 blocked checks.
   The ordinary serving proof here does not close those acceptance cases.
