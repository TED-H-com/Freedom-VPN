# Isolation run procedure — Follow-up 16

Product baseline **`9efe9773e67d35839649e9f21e1c3a9b680633b2`**, including #392
and devstack 27 (#394). The final `go env GOCACHE` query returned
**`$HOME/.cache/go-build`** (the home prefix is redacted for repository hygiene;
raw output is retained privately). The pre-run note recorded
`/data/ted-h-cache/go-build`, but the assembled log retained no separate runtime
cache receipt. The discrepancy is unresolved; the run-time cache path is
**unproven**. The runner inherits GOCACHE and never hard-codes or overrides it.
The measured consumer commit and
run outcome are recorded below and in [summary.md](summary.md).

D-098 first commit **`3005c383`** copied all **123** root files and subtrees of
run 0b9e4510 byte-identically to [follow-up 15](history/follow-up-15/archive.json).
All files were compared to baseline Git blobs; gzip pins include both forms.
Follow-up-15-stream and every earlier archive stay unchanged. Handover claims
already read immutable archives; its living-root strings are rejection inputs.
No D-132 exception edit is required. Existing files remain accessible until
fresh replacements are published; historical links use immutable snapshots.

## Predeclared D-140 decision

Exactly **one whole-node sweep** in one assembled run. The existing driver
has one whole-node expiry boundary; the separate finite-member expiry is not
another whole-node boundary. Repeating the lifecycle to phase another sweep
would extend the change beyond the boundary schedule. No retries and no merged
traces. `EvaluateDeadline`, its tests, `phased` and `arbitrary_37ms` are unchanged.

Calibrate from this run's original cadence, copy passive capture at D−3 s,
and plan at D−1 s. Use at least 20 samples and
`ceil_0.1ms(max(0.5ms, spread + 2*max_transit + 0.1ms))`. Record all inputs,
scheduled and actual sends, ingress and egress. Anchor the added grid at D−100 ms,
extend it beyond [D−101 ms,D+100 ms] by spread plus three spacings, and omit
added sends within one spacing of either unchanged original cadence.

**The trade-off is declared before running:** one sweep cannot guarantee both
noninterleaving and a deciding probe in the roughly 3 ms historical positive
window. Prior calibration would choose 4.3 ms; a grid that coarse can miss it.
The unchanged D−101 ms original send predicts ingress at D−101 ms plus this
run's median latency. It can land in the window if latency and closure permit,
but neither prediction nor historical closure is a guarantee. The added anchor
can also be omitted by the original-cadence exclusion. Qualification checks
actual ingress→egress ordering; the unchanged evaluator decides the result.
A qualified trace missing the requisite positive **fails**, even if lack of
resolution is plausible. Run 56d20530 isolation F14-01 and run 0b9e4510's
3.821 µs interleaving stay failed; all losses and attempts remain counted.

Only member sends and passive packet captures observe this boundary. No node
ruleset observer is added. Existing passive netlink receipts are supplemental.

## Populated incidents

Before any member revocation, require a latest accepted serving report joined
to its matching desired/applied/signed generation, at least two leased members,
and more than 50 seconds remaining on the unchanged 60-second TEST ONLY lease.
Use devstack 27's `ArmMatchingLeaseHold` once per incident, each a new UUID and
a 45-second TEST ONLY bound, followed by a genuine ETag/idempotent capacity PATCH.
No signed body or report is synthesized, edited or replayed. The existing owned
observer now forwards to the **unchanged shared proxy**, so that implementation
owns withholding/release; observation stays downstream of the seam.

First incident: retain alarm and single escalation, independent packet denial,
temporarily move the node container's conntrack executable to cause an unrelated
failure, restore it, then supervise an agent-only stop/start while the proxy
remains armed. Read all incident snapshots and local counters. Release before
the bound when possible, join the fresh forwarded lease to the local digest and
accepted matching report, and separately prove counted serving recovery.
Only after recovery, arm a second capacity transition and join its distinct
incident, alarm, escalation and matching recovery. A missing row remains
**not exercised**. These explicit attempts supplement the earlier single
opportunistic control-cut row; they do not replace or retry it.

## Foreground commands and host admission

```sh
make -f mk/at-isolation.mk at-isolation-lint at-isolation-test at-isolation-build
python3 -B tests/e2e/at/isolation/host_preflight.py
pgrep -af 'gates.sh|mk/at-'
go env GOCACHE
make -f mk/at-isolation.mk at-isolation
python3 -B tests/e2e/at/isolation/review_boundary.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_policy_missing.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_reservations.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_core32.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_holds.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_populated_holds.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_commands.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_stops.py docs/e2e/at/isolation
python3 tests/e2e/at/isolation/run.py go test -count=1 ./tests/handover/...
python3 tests/e2e/at/isolation/run.py make go-lint go-with-disk-tmp=
test -z "$(gofmt -l $(git ls-files '*.go'))"
git diff --check
python3 tests/e2e/at/isolation/run.py go test -count=1 -v -run '^TestCommittedEvidenceFiles_ValidAndHygieneClean$' ./tests/e2e/at/isolation
```

The preflight records `vmstat 5 7`, load averages and suite inventory, repeating
for up to 30 minutes until all six current rows have r≤3 and id≥80; the first
since-boot row is ignored. At the limit, run and disclose the host state. This
is context, never a pass criterion. Private process commands stay in `.agent`;
public evidence records PIDs and suite kinds without private paths.

`run.py` supplies disk TMPDIR/GOTMPDIR under `.agent/tmp`, waits while two other
suite trees are active, and rechecks after each stack startup. It reports the
wait. No shell cleanup trap. Root lint overrides only its trap wrapper, retaining
the same formatting/vet/pinned-staticcheck commands. Docker names stay under
`fvpn-w4-isolation-*`; teardown requires containers/images/networks/volumes zero.
Stop receipts retain actual agent exit/signal and separate PID1 outcomes.
AT11 native platforms and AT13's full fault matrix remain carried operator rows.
Large JSON is published as lossless gzip with both-form SHA-256 pins only at
immutable history paths. Older archives are never rewritten.

## Recorded outcome

Run **83b457ef-ea16-490b-9923-1fea0df86b36**, consumer
**a8394027002771be64f023d062ea47b2da1447d8**, source dirty **false**.
Started **2026-11-01T16:54:27.108813Z**, full foreground duration **1406.378 s**,
exit **2**; driver **1272.20 s**. This was the only assembled attempt.
Host became quiet after **1561.452 s**, within the 30-minute bound, with 52
vmstat windows retained. Suite checks saw no other suite roots and waited
0.084/0.088/0.092/0.129 s respectively. All owned Docker inventories are zero.

103 calibration packets selected **2.2 ms**, yielding90 dense plus202 unchanged
cadence probes. The unchanged evaluator passes; D-140 qualification fails one
**3.609 µs** overlap. F16-01 separately retains the shared D-104 fixture failure
before trace production. Two populated incidents cover all seven requested
categories. Five agent parent receipts record exit 0. AT11/AT13 remain blocked
on their carried operator matrices. No failed row was retried or reclassified.

Publication uses12 immutable lossless gzip streams with compressed and expanded
SHA-256 pins. Six capture text files drop only surplus terminal blank lines, and the D-104
assertion excerpt drops trailing whitespace, to
satisfy diff whitespace checks; original capture bytes remain in private run
artifacts. The inherited D-104 trace and appended mechanism-summary prefix are
preserved in follow-up 15, not presented as fresh observations. Only offline hold
review scoping and evidence/documentation changed after the measured consumer.
