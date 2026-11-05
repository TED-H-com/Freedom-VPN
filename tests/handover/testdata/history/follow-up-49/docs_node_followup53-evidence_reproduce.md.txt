# Follow-up 53 evidence and reproduction

This directory preserves measured attempts, including failures. `commands.json`
records new command arguments, individual exits, tested HEAD and working-tree
patch identity, counts and sanitized logs. The two initial sixth-failure attempts
have separate JSON records. `ci-metadata.json` records read-only GitHub metadata;
`ci-failures.txt` contains the actual failed assertions from the six logs.

Counts distinguish parent tests and subtests; those totals overlap by nesting.
A package pass does not mean every test ran: skips and no-test packages must be
read separately. Non-verbose plain module output does not record per-test counts.
Package output time and supervisor elapsed time are separate measurements.

The `.patch` files are snapshots of tested source, not current design documents.
The before-fix meter overlay uses `b921c75f`'s meter.go and delta.go; the R23
negative control uses the same overlay. The R29 synchronous negative controls
are described individually in commands.json, including the first attempt's
blocked teardown and missing replacement hash. No failed attempt is replaced
by a later success. Historical aggregates in the dated note are corroborated
transcript excerpts; the old raw timing files are gone and are not in this
folder. The stopped earlier scratch is not their source.

Run from the worktree root with the launcher's unique disk TMPDIR/GOTMPDIR under
`~/agent-work/tmp/freedom-vpn/`. Verify the filesystem with `findmnt -T "$TMPDIR"`;
never use host tmpfs. The supervisor inherits those directories, sets
CGO_ENABLED=0, GOTOOLCHAIN=local and empty GOFLAGS, and invokes the installed
project Go shim. It checks `$FVPN_HOLD_SH` and the suite-overlap process patterns
before starting. Status 125 means not admitted and is not a test result.

Example for a new, uniquely named reproduction attempt:

```sh
python3 -B docs/node/followup53-evidence/run.py reproduction-meter 24 \
  go test -json -count=50 -timeout=20m \
  -run '^(TestCheckpointForRestartAtSameMicrosecond|TestMTR03_(RestartCheckpoint.*|FinalSampleUnchangedRollback))$' \
  ./node/internal/meter/
```

The supervisor refuses to overwrite a recorded attempt. It starts bounded CPU
worker children only after admission, explicitly terminates and joins them,
and preserves the command's own status. No shell cleanup trap or host slot
reservation is used. Generated private logs go under `.agent/f53-fix1/`.

For gate names beginning `gate-`, the supervisor installs temporary make/docker
admission wrappers and a node-go wrapper under the same disk TMPDIR. Admission
messages are captured separately in `.agent/f53-fix1/child-admission.jsonl`;
node-go adds `-json` only to `go test`, for recorded counts. Build, vet and
staticcheck arguments otherwise pass through. Host Python subprocesses receive
`-B`. Recursive make and Docker calls retain admission and disk settings.

Every node make invocation must pass `NODE_TMPDIR="$TMPDIR"` on the command
line. The lab additionally passes NETLAB_SCRATCH and NETLAB_BIN below TMPDIR,
`netlab-disk-env=env`, `FVPN_AGENT_EVIDENCE_DIR` below TMPDIR and the owned prefix
`fvpn-w2-agent-core-f53-fix1`. The root go-lint command overrides
`go-with-disk-tmp` with `set -eu;` so the existing launcher directories are reused
without the recipe's shell trap. The completed gate ledger records the actual
arguments. Do not rerun a failed lab scenario; the assignment permits only one
pre-scenario harness/setup rerun, and only when that condition is demonstrated.

Public logs replace the worktree, launcher directory and queried Go cache path
with `$WORKTREE`, `$TMPDIR` and `$GOCACHE`. These are redaction labels, not invented
filesystem locations. `go env GOCACHE` was queried; no private cache path is
published. Test-generated IDs in logs are observations of fixtures, not real
infrastructure identifiers. Source manifests and patch/log hashes are computed
from the recorded bytes.
