# Follow-up 26 reproduction (TEST ONLY)

Use only the assigned worktree and `fvpn-e2e-*` inventory. Before startup, check
`pgrep -af 'gates.sh|mk/at-'` and wait while two other integration suites are
running; check again after startup. A reset is required for **each** attempt.
Keep failed admissions, failed requests and nonzero test exits.

```sh
mkdir -p .agent/tmp/followup26
export TMPDIR="$PWD/.agent/tmp/followup26"
export GOTMPDIR="$TMPDIR"
export GOCACHE="$(go env GOCACHE)"
export GOTOOLCHAIN=local
export CGO_ENABLED=0
make -f mk/e2e.mk e2e-up E2E_RESET=1
env FVPN_E2E_FOLLOWUP26=1 FVPN_E2E_FOLLOWUP26_FAMILY=ipv4 \
  go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e \
  -run '^TestE2E_Followup26Native$'
make -f mk/e2e.mk e2e-down
```

Replace `ipv4` with `ipv6` for the other family. The default omits the unsafe
node tracer and proves serving after observer setup. To reproduce the historical
observer, explicitly add `FVPN_E2E_REPRODUCE_TRACE26=1`; its safety failures are
diagnostic results, not transport acceptance. The separately declared hold
experiment also sets `FVPN_E2E_TRACE_HOLD26=1`, which observes twelve one-second
intervals after installation before selection. It intentionally exposes an
admitted node to the unsafe observer and keeps the resulting failed request.

The native runner exports central history and private artifact receipts before
teardown. Its serving gate has a 90-second bound; failure launches no original
request. D-077 re-grant starts only after a counted first flow, captures before
fencing, observes withdrawal, re-authorizes with an idempotent replay, proves
populated serving and sends one original request without a reconnect or retry.
A failed first flow keeps a separate unattempted re-grant prerequisite row.

On an enrolled stack, before teardown, the stop proof is:

```sh
env FVPN_E2E_STOP26=1 go test -tags e2e -count=1 -v -timeout 5m \
  ./tests/e2e -run '^TestE2E_ContainerStop26$'
```

It leaves the node container stopped. Run normal `e2e-down` afterward. The first
case requires parent-observed agent exit 0 and Docker exit 0; the second verifies
automatic resume, SIGSTOPs the new agent, and requires its signal-9 wait receipt
and a supervisor escalation after at least 15 seconds. Docker's configured
25-second grace is used. Container exit 0 by itself is insufficient.

For a **new unsealed** run, derive joins once from its private receipts:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 tests/e2e/analyze_followup26.py \
  docs/e2e/runs/2026-11-01/followup26/<run-id> \
  .agent/tmp/followup26/<run-id>
```

The analyzer refuses an existing join. `--session-only` adds the separately named
session supplement to an older analyzed run without rewriting original joins.
Never rerun a writer over a sealed archive. Full commands, order, actual exits
and output tails for this campaign are in [gates.md](gates.md); all original
attempts are in [campaign.json](campaign.json). Private raw bytes are not copied
into the public archive. The pin manifest hashes only immutable archive paths.
