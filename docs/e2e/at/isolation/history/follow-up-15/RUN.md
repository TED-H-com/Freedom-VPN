# Isolation run procedure — Follow-up 15

Product baseline **`aa7f306c008d8021b3921d083b70e19db7c8f865`**, including
isolation 14 (#383), devstack 26 (#386), handover 27 (#387) and agent-core 34 (#388).
Runtime consumer **`9031ebd4`**. `go env GOCACHE` returned
**`/data/ted-h-cache/go-build`**; no cache path is hard-coded in commands.
Offline consumer changes after runtime compilation are separate from the measured binaries.

D-098 first commit **`877ea4e3`** copied all **117** run-56d20530 root files,
including subtrees, byte-identically to
[follow-up 14](history/follow-up-14/archive.json). Every byte was compared with
the baseline Git blob; all compressed/decompressed forms have SHA-256 pins.
Earlier archives, including `history/follow-up-14-stream/`, are untouched.
Handover claims already read immutable snapshots; living-root strings are
rejection inputs. No D-132 one-file exception edit was needed.

Run from the worktree root, in the foreground:

```sh
make -f mk/at-isolation.mk at-isolation-lint at-isolation-test at-isolation-build
vmstat 5 7
pgrep -af 'gates.sh|mk/at-'
go env GOCACHE
make -f mk/at-isolation.mk at-isolation
python3 -B tests/e2e/at/isolation/review_boundary.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_policy_missing.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_reservations.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_core32.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_holds.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_commands.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_stops.py docs/e2e/at/isolation
python3 tests/e2e/at/isolation/run.py go test -count=1 ./tests/handover/...
python3 tests/e2e/at/isolation/run.py make go-lint go-with-disk-tmp=
test -z "$(gofmt -l $(git ls-files '*.go'))"
git diff --check
python3 tests/e2e/at/isolation/run.py go test -count=1 -v -run '^TestCommittedEvidenceFiles_ValidAndHygieneClean$' ./tests/e2e/at/isolation
```

`run.py` provides disk TMPDIR/GOTMPDIR under `.agent/tmp`, explicit teardown
without shell cleanup traps, and suite-tree admission before each stack and
after startup. It waits while two other suite trees are active. The root lint
override removes only its temporary-directory shell trap; the same gofmt, vet
and pinned staticcheck gates run. Only `fvpn-w4-isolation-*` objects are owned;
teardown must leave containers, images, networks and volumes at zero.

[Preflight](host-preflight.json) retains all seven `vmstat 5 7` rows and load
averages. The six interval rows have r=0–4 and id=83–90%; **not quiet** because
two rows have r=4. The first, since-boot row has id=79%. Host state is context,
not a criterion or an explanation for a failure. Exact process output remains
private; public process entries retain only PID and suite kind.

## D-140: measure first, add a boundary schedule

`EvaluateDeadline` and its tests remain byte-identical. It requires a positive
leaving at or after D−101 ms, no positive after the first drop, and first drop
by D+100 ms; egress at/after D also fails. A straddling closure interval remains
failed. Historical **isolation F14-01** in run 56d20530 remains failed and is
distinct from **members F14-01**. No attempt or loss is waived.

The original `phased` and `arbitrary_37ms` sends are unchanged. At D−3 s the
driver copies the existing passive node packet capture to the member. At D−1 s,
`deadlineProbes` joins that capture to its own completed original-cadence sends.
It records every calibration sample and the minimum, median, maximum and spread
of member-send → ingress capture-read latency. At least 20 samples are required.

Added probes use median latency to predict ingress. Predicted ingress spans
D−101 ms minus the latency spread and three spacings, through D+100 ms plus
that margin. The measured binary used max(0.5 ms, twice the largest measured ingress→egress
transit plus 0.1 ms), rounded upward to 0.1 ms. That selected 0.6 ms and
produced one interleaving, so this attempt is not D-140 qualified. The final
source adds the measured latency spread to the spacing expression; it would
select 4.3 ms from this calibration. This correction was made after the run
and has no assembled qualification yet. The sample remains unchanged. Added sends within one spacing
of an original scheduled send are omitted; original sends never move. A short
member-side scheduling spin avoids timer coalescing for added probes only.
Actual scheduled/send/ingress/egress times and every transit are retained;
[qualification](boundary-review.json) checks actual ordering and reports any
positive whose egress interleaves the next ingress. Prediction is not proof.
The unchanged Go evaluator alone supplies the packet verdict. The final runner
also runs the independent schedule qualifier; its nonzero exit retains an
interleaving failure without editing that verdict. This wiring was added after
the measured run. The sole attempt’s offline qualifier exits 1.

Observation uses member sends and passive AF_PACKET captures. The existing
read-only `nft monitor` receipts may accompany producer evidence but never
replace packet evidence. No node nft table/rule or `nftrace` statement is added.
Devstack 26 established that a foreign tracer can itself close the node under
the unchanged verifier; this observer leaves the ruleset unchanged.

## D-102 stops and core 34 reservation

Before the original whole-container restart and clone-source stop, retain
launch ID, PID and `/proc` start ticks. The ordinary stop explicitly gives the
container 25 seconds; devstack's clone wrapper also gives 25 seconds. The
supervisor's agent budget and direct pidfd sampler use **15 seconds**.

After the whole-container stop, use `docker cp` for the matching
`agent-exits/<id>.json` before restart. The clone source uses the same receipt
copy after its original fixture stop/start. The direct sampler reads the
parent receipt after the service stop. Each checks all three identity fields.
The parent's exit code/signal is the agent result; container exit records only
PID1 handler completion. `.sigkill.json` records attempted escalation, never
its outcome. Historical 10.087 s and 20.153 s exit-137 rows remain as recorded.
Step cancellation time is not exported and is not inferred.

[Reservation timelines](pre-apply-reservation-review.json) read
`host.before_apply.end_ns` → `timing.start_ns` for every Apply, including failed
Applies. For Steps with original conntrack revoke commands, this bracket contains
the durable reservation plus small common setup before the restricting install.
Receipt-to-revocation and baseline-to-recovery retain that elapsed time, outside
the guard percentiles. Desired response completion is a proxy delivery receipt,
not an exact node parse timestamp; producer wall/monotonic anchor bounds are
preserved. A pure renewal/expiry Step with no queued deletes adds no reservation.
A failed preparation without delete commands cannot prove the queue was empty.

## Retained rows and limits

D-087 uses the exported seam, fixed generation and pre-enrollment 60-second
TEST ONLY fleet setting. ISO-F6-01 keeps its 90-second readiness budget, twelve
samples at 500 ms and all six assertions. AT10–AT13 retain all previous normal
and failure paths. AT11's native platform matrix and AT13's full fault matrix
remain carried operator rows, including physical arm64 and kernel reboot.

The renewal diagnostic retains every loss, including run 56d20530's packets
15–18 outside guards in the 30→31 baseline bracket. D-120's bounded allowance
applies only to its separate mechanism stream. Completed guards are the next
independent distribution sample; no historical ceiling or A/B attribution is
invented. All original per-Apply command indices and failed commands remain.

Every accepted stale-lease hold is joined to wire-nodes 34's central report,
lease, sample and incident; poll-time desired state is separate from delivered
desired state. Populated hold: **not exercised — devstack 27's fixture pending**.
No alarm, escalation, active-stop retention or populated packet proof is inferred.
The previously authorized single existing-seam attempt remains retained; it is
not the pending populated-hold fixture.

Large JSON/JSONL is published losslessly as gzip, with both-form hashes only
at immutable `history/` paths. Offline consumers accept either form. Markdown
links remain navigable, and historical archives are not recompressed.
