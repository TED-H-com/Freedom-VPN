# Isolation run procedure — Follow-up 14

Product baseline **`9dde5eeb60a6cb6b9b10974c3100ef1ea1352698`**, including
isolation 13 (#380), wire-nodes 34 (#382) and netlab 23 (#377).
Runtime consumer **`716a16335cb5335473416740c76a75ce95536796`**.
`go env GOCACHE` returned **`/data/ted-h-cache/go-build`**. The runner queries
Go's environment; it does not hard-code that path.

D-098 was the first commit, **`5e66101c`**: all 112 living root files from run
`38074149-1137-4782-80fd-8669c36c6fcb` were copied byte-identically to
[history/follow-up-13](history/follow-up-13/archive.json), with SHA-256 pins.
The new archive manifest uses the same compressed/decompressed fields as
follow-up 12. Earlier archives are untouched. Handover's living-root strings
are rejection-test inputs, so the D-132 one-file exception was not needed.

Run from the worktree root:

```sh
make -f mk/at-isolation.mk at-isolation-lint at-isolation-test at-isolation-build
vmstat 5 7
pgrep -af 'gates.sh|mk/at-'
go env GOCACHE
make -f mk/at-isolation.mk at-isolation
python3 -B tests/e2e/at/isolation/review_policy_missing.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_core32.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_holds.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_commands.py docs/e2e/at/isolation
python3 tests/e2e/at/isolation/run.py go test -count=1 ./tests/handover/...
python3 tests/e2e/at/isolation/run.py make go-lint go-with-disk-tmp=
test -z "$(gofmt -l $(git ls-files '*.go'))"
git diff --check
python3 tests/e2e/at/isolation/run.py go test -count=1 -v -run '^TestCommittedEvidenceFiles_ValidAndHygieneClean$' ./tests/e2e/at/isolation
```

All gates run in the foreground. `run.py` provides disk TMPDIR/GOTMPDIR under
`.agent/tmp`, uses explicit teardown without shell cleanup traps, checks suite
process trees before each stack and after startup, and waits while two other
suites are active. Only `fvpn-w4-isolation-*` Docker objects are owned. The
final inventory must show zero containers, images, networks and volumes.
The root lint override changes only the temporary-directory wrapper; the
pinned gofmt, vet and staticcheck gates remain the root gate.

[host-preflight.json](host-preflight.json) records the seven `vmstat` rows and
sanitized process inventory; the exact original stdout is retained privately
under `.agent/f14`. This is host context, not attribution or a quiet-host claim.
The run's source identity remains fixed while other lanes advance main.

## Unchanged acceptance measurements

D-087 runs first on `FVPN_E2E_GRANT_SEAM=1`: initial counted forwarding, 65
seconds of stable invariants, false/true eligibility, natural signed leases,
original wire bytes and actual accepted reports. The fleet's TEST ONLY
60-second offline window is set before enrollment and is unchanged thereafter.
No generation forcing, projector stop, transport fault or manufactured report
is used for D-087.

AT10–AT13 retain protected destinations, peer/management/DNS/source-binding
checks, publication groups, revoke, ordinary restart, finite-member expiry,
whole-node D-104 expiry, the 70-second renewal diagnostic, ISO-F6-01 and the
identity-freeze fixture. AT11's native platform matrix and AT13's full fault
matrix remain operator rows. Physical arm64 and kernel reboot are not exercised.

D-120 keeps its original stream, verdict and allowance: at most five eligible
losses, each with ingress in exactly one verified guard, guard and
render-to-install within 100 ms, the next datagram succeeding, and zero TCP
timeouts. That allowance never applies to the separate 80 ms diagnostic.
Every renewal/expiry loss remains counted. D-104 keeps `[D−101 ms,D]` and
rejects positives at/after D; no cadence allowance or timeout inflation.

ISO-F6-01 uses the existing 90-second restored-control readiness budget and
maximum twelve dirty samples at 500 ms. Assertion 6 requires a current accepted
`apply_candidate` rejection joined to desired wire and original failed Apply
commands. Installed-verifier requirements retain separate original evidence.
The six original assertion expressions are unchanged.

## Central generation holds and the one control-cut attempt

The owned driver polls accepted observations read-only throughout the run.
`generation-holds-central.json` retains each original report and observation
row ID, its own signed lease, decimal applied/lease generations, the central
desired generation at read time, per-report sample, incident, and final audit
history. `PublicationDataPathFailure` evaluates genuine reports against the
live database. A read-only replay reverses their checks/errors to test order
independence; no replay is ingested. Poll-time values are labelled separately
from receipt-time values.

`publication-operation-history.json` retains actual operation steps and
`observed_refs`. In this run the owned read-only sidecar was started during the
assembled driver, before interface-change completion, and made a final read
after central export. Initial already-completed operations are retained state,
not continuously observed earlier transitions. The final runner starts this
same sidecar and the Docker lifecycle observer before the driver on future invocations; that startup wiring was
added after this run's binaries started and is not retroactively claimed here.

After the unchanged D-104 window, the harness makes exactly one public capacity
change. For at most 15 seconds it looks for a new accepted open hold, then uses
the existing Docker control/worker pause boundary. If the node has already
renewed, the attempt is labelled not exercised; it never retries or adds a
lease interception hook. If the old lease is still held, it records independent
packet denial and a kernel snapshot, holds control loss for 35 seconds, invokes
the existing pidfd terminal-stop helper, reads durable incident state, restarts
through the existing supervisor, and restores control. All original bodies,
local counters and escalation IDs decide the outcome. No alarm is inferred
from elapsed time or forwarding denial.

`generation-holds-review.json` joins every accepted hold and every populated
old/new producer window. A window without a reported hold retains that gap;
`policy_missing` alone never creates a central incident. Matching-lease endings
join the exact receipt time and report ID. A later incident must have a distinct
ID. A kernel reboot is **not exercised**, under AT13's carried operator matrix.
The original terminal source/clone stop samples remain separate from this
control-cut attempt and from Docker PID1 SIGKILL/exit-137 events.

## Producer commands, recovery and counter-read failures

The full original producer stream retains all processes, sequences, failed
Applies, command indices, verifiers, baseline writes and heartbeats. Completed
guard distributions are a new independent sample after follow-up 13, not A/B
attribution or a new ceiling. Netlab 23's `skip_reason=install_bound` is retained:
resource sampling is unavailable in that phase, while command timestamps,
exit/signal and process CPU/switch evidence remain separate. The offline reader
accepts the documented skip kinds and rejects a skipped sample carrying values.

D-135 / corrected D-136 retain baseline→next successful Apply and later
verification/heartbeat evidence. Only retained Apply-error recovery uses the
verified deny-all candidate. Neither the 100 ms install budget nor a completed
baseline bracket supplies a recovery SLA. Unsampled packet intervals stay
unavailable, never zero-loss.

At every counter-read failure, apply
[netlab 23's decision](../../../poc/G2-followup23-evidence/decision.json):
`*enforce.DuplicateChainError` / `duplicate_chain` denotes an ambiguous owned-chain
listing. The [0 / 3,300 experiment](../../../poc/G2-followup23-evidence/reproduction-summary.json)
did not establish its origin. The shared counter reader may still publish a
generic `parser_rejected`; this lane does not relabel an untyped historical
failure. A failed read never proves counters frozen, counters removed, or
forwarding closed. No retry waiver, acceptance change or historical
reclassification is made.

Large producer JSONL is published as lossless gzip. Immutable `history/` pins
record SHA-256 and sizes for compressed and decompressed forms, independently
compared to the original export. Living root paths are never evidence pins.
All historical failures and unproven causes remain in their original archives.
