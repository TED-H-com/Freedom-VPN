# Isolation run procedure — Follow-up 17

Baseline **b943941157ab6aeba6b04c79377de686c22bfbb4**, including isolation 16
(#399), netlab 26 (#403), and agent-core 36 (#401). Agent-core 36 installs signal
handling before open/Boot/Start and retains it through Close. New stop receipts
will be reported without reinterpreting historical outcomes.

D-098 first commit **fbe2e789** archives all **129** root files and subtrees from
run 83b457ef byte-identically with SHA-256 pins at
[follow-up 16](history/follow-up-16/archive.json). All **1,378** earlier archive
files, including follow-up-16-stream, are untouched. Handover claims already read
immutable paths; no D-132 exception edit is needed.

## Predeclared D-140 schedule and D-142 scope

Exactly one whole-node expiry sweep in one assembled run; no retry. Retain the
corrected follow-up 16 schedule: 101 phased and 101 arbitrary_37ms original
probes, same-run calibration capture copied at D−3 s, planning at D−1 s from
at least 20 original-cadence packets. Let spread=max−min member-send→node-ingress
latency and transit=max ingress→egress latency. Choose spacing
`ceil_0.1ms(max(0.5ms, spread + 2*transit + 0.1ms))`. Record every input and the
chosen spacing before the boundary. Anchor predicted added ingress at D−100 ms;
subtract median latency for sends. Extend beyond [D−101 ms,D+100 ms] by spread
plus three spacings; exclude added sends within one spacing of an original send.

**A predicted ingress can land in [D−101 ms, closure).** The original D−101 ms
send predicts ingress at D−101 ms plus the measured median; the added anchor
predicts D−100 ms if not excluded. Actual closure and future latency are unknown.
One sweep cannot guarantee resolution within the historical roughly 3 ms positive
window as well as separation. A closure interval straddling D−101 ms still fails.
No missed positive is waived.

**D-142 applies only to this new sample.** Order probes by observed ingress.
Judge each adjacent pair starting with whichever comes earlier in that order:
the last positive or the first drop, continuing through the end of the sweep.
Each positive egress must precede the next ingress. Record earlier
positive-positive overlaps with both packets' send, ingress, egress and latency,
but do not use them to disqualify this sample. Retain deciding-pair offsets and
latencies separately. `EvaluateDeadline` and its tests are byte-identical and
own the verdict; qualification never promotes an evaluator failure. Historical
isolation 15, isolation 16 and isolation F14-01 remain failed as judged.

Observation is member-side sends and passive packet capture only. No node rule
is added. The separate D-104 mechanism test consumes netlab 26's changed precision
fixture, read-only. Its failure, if any, is a new retained finding; no retry.

## Hold attempts

Use only devstack 27's D-102 arm/capacity-command/release seam. Preserve the first
two incident sequences and all their joins: two old-lease members, new arm UUID,
45-second TEST ONLY bound, real ETag/idempotent capacity PATCH and read-back;
first incident alarm, unrelated conntrack-executable failure and restoration,
agent-only stop/reset, genuine matching lease/accepted report and counted recovery;
then a distinct second alarm incident after recovery.

Third explicit incident attempts central's 30-second fallback: establish a fresh
populated lease with >50 seconds remaining, arm the same 45-second seam, issue the
capacity command, stop the agent after the first accepted hold report, retain
proxy state and central snapshots, and wait up to 35 seconds for the original
audit. Restart and release, joining genuine matching recovery and counted traffic.
An earlier reported alarm does **not** prove fallback. No report is suppressed,
edited, synthesized or replayed. If the required row is absent, record it as not
exercised and name the missing capability. Inspect all original hold reports for
pure `stale_generation`; a dependent QoS check classified as `check_failed` is
never relabelled.

## Foreground gates and host admission

```sh
make -f mk/at-isolation.mk at-isolation-lint at-isolation-test at-isolation-build
FVPN_AT_EVIDENCE_DIR=.agent/f17/evidence python3 -B tests/e2e/at/isolation/host_preflight.py
pgrep -af 'gates.sh|mk/at-'
go env GOCACHE
FVPN_AT_EVIDENCE_DIR=.agent/f17/evidence make -f mk/at-isolation.mk at-isolation
go env GOCACHE
python3 tests/e2e/at/isolation/run.py go test -count=1 ./tests/handover/...
python3 tests/e2e/at/isolation/run.py make go-lint go-with-disk-tmp=
test -z "$(gofmt -l $(git ls-files '*.go'))"
git diff --check
python3 tests/e2e/at/isolation/run.py go test -count=1 -v -run '^TestCommittedEvidenceFiles_ValidAndHygieneClean$' ./tests/e2e/at/isolation
```

Fresh evidence directory prevents inheriting an old packet window or appended
mechanism summary. Public cache receipts use `$HOME` for the home prefix. The
runner inherits GOCACHE; it uses disk TMPDIR/GOTMPDIR below `.agent/tmp`, with
explicit cleanup and no shell traps. Root lint bypasses only its trap wrapper.

Before the run record load average, `pgrep -af 'gates.sh|mk/at-'` and `vmstat 5 7`.
Ignore vmstat's since-boot row; require r≤3 and id≥80 in all six current rows.
Wait up to 30 minutes for quiet, then run and disclose it if still busy. Host
state is context, not a pass criterion. Independently wait up to 90 minutes for
real members assembled, agent-core node-test-netlab, or w1-netlab tagged netlab
suites to finish. Shell text merely mentioning commands is not a run. Never
start with a conflicting timing-sensitive suite; stop that item if the deadline
expires. Check suite count before and after every stack startup; wait while two
other suite trees are active. Record all admission waits.

Only `fvpn-w4-isolation-*` Docker objects are owned. Tear down explicitly and
record zero containers, images, networks and volumes. No host network changes.

## Scheduled nightly policy

[Expected blocks](../../../../tests/e2e/at/isolation/expected-blocked.json) list
only the established AT11/AT13 operator rows. D-144 changes the scheduled exit
mapping only. Listed blocked rows retain their status, reason and owner in JSON
and job summary; a passing listed row is reported as unblocked for list removal.
Any other block, fail, subprocess error or traceback remains red. PR runs are
unchanged. A fresh records-only artifact is uploaded with seven-day retention;
stack state, credentials, cookies and raw workflow logs are excluded.

Both historical nightly logs were read using `gh run view --log`. They show the
short whole-node expiry blocked with positive=false and no boundary execution;
neither prints its actual reason. The observed row is not established as an
expected runner limitation and is not allowlisted. Future row logs include the
recorded reason/owner. The Nov 1 log actually failed in the producer review
(`invalid lifecycle resource block`); Nov 2 reached the boundary ValueError.
Both reviews now record `blocked: expiry not executed`, refuse inherited evidence,
and return nonzero with one line before reading any old producer/packet file.
No workflow is triggered by this task.

AT11 native platforms and AT13's full fault matrix remain carried operator rows.

## Recorded outcome

One run **0ec14b39-467e-4a37-a239-1a33ef12eb71**, measured consumer
**de9682a96a296a3b79686db029b3e28f493f08f4**, source dirty false. Started
**2026-11-03T06:16:01.539081Z**, ended **06:41:05.602278Z**; full **1504.063 s**,
driver **1338.92 s**, exit **2**. GOCACHE before and after:
**`$HOME/.cache/go-build`**. Host quiet after **60.040 s**; one post-start
conflicting tagged netlab test required **30.078 s** waiting. Inventory zero.

The 103 calibration packets selected **2.4 ms**. All **285** probes are retained,
with **zero overlaps**. D-142 qualification and unchanged evaluator pass;
last positive **D−98.970225 ms**, first drop **D−95.769511 ms**. D-104 mechanism
passes; all original seven hold categories and central-age fallback pass.
Pure classification remains not exercised. AT11/AT13 remain operator-blocked.
Six agent exits are 0; no install overrun. [Full result](summary.md).

Final offline-only fixes scope counter-reset review to its planned incident and
remove a fixed-depth assumption in evaluator metadata lookup. The measured
metadata omission and original offline error remain in
[consumer corrections](consumer-review-corrections.json). No second measurement.
