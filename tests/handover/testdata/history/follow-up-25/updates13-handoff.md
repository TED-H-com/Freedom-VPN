## Summary

Updates follow-up 13 (session launched 04:33 EST on 2026-11-01 when #369 landed; 4 commits, 106 owned files, merge-tree clean, prepush-scan clean). The regression run for agent-core 31's shutdown change (#369) and netlab 22's opt-in observation (#365, not enabled in this lane's configs). No product change. A pre-test stack-start failure (an unrelated host listener on port 42443) is retained under `history/follow-up-13-startup/`; the owned harness gained a configurable port slot (default unchanged) and the single regular AT invocation ran on slot 54000.

## Summary

Final commit `d54b3a4b90de1f5aa19d870baeca1eea1beff2d2`; measured harness `20d1f6e83d4a0dce97e2cf87e4ccd4a31dc580a5`;
product main `5bd751026073444087b32ac2a7115f8566104f8b`. Run `759d244a-2dc4-460c-9f82-e37ce0ac7560`
passes regular **AT23–AT27 5/5**, installer **27/27**, shared-wrapper
**installed / exit 0**, 13 result POSTs, physical AT26 rollback, and the
uncached-fallback probe. The live target exits **0**; Go test **517.660 s**.
There was **one regular Go AT invocation**. F11-01/F11-02 stay CLOSED and
F18-01 stays the intended stopped-producer negative. Requests: **1 (F13-01)**.

All 11 updater stop commands were timed directly and returned success, with no
command timeout or lost stored lease observed. Five stopped agents have captured
normal exit status 0, five calls find the agent already stopped, and AT23's first
agent termination is **unobserved** because it was reaped before the 1 ms sampler
saw its wait status. No claim that every agent exited gracefully is made. The
complete termination measurement is therefore incomplete; AT predicates remain
independently passing. F13-01 requests reliable fixture exit-status export from
devstack; no product failure is inferred and no retry was made to fill the gap.

The requested historical direct comparison also has a hard evidence limit:
run 2babf9b6 never captured stop-command durations or process exit statuses.
Its retained regular journals yield all nine larger stage intervals, but the
installer journal was not retained. Missing direct measurements stay missing,
not replaced by stage intervals or a later rerun.

D-098 first: commit `6f1eb10c` copied all **44** Follow-up 12 root files
byte-identically from the source main blobs into `history/follow-up-12/`, with
SHA256SUMS in the Follow-up 11 shape, and extended D-132 before any gate or lab.
All 427 pre-existing archive files remain byte-identical, including
`follow-up-12-first-attempt`. A separate immutable retained-journal projection
has four transaction IDs, all matched to the archived run, and contains only
transaction/stage/sequence/time fields.

One stack-start failure is retained in `history/follow-up-13-startup/`:
make exit 2 in 24.395 s because an unrelated host listener occupied 42443.
No regular test or installer test started. The original note proposed waiting;
the final owned-harness decision made the port slot configurable, kept default
42000, and selected shared-harness-supported slot 54000 after checking its ports.
No foreign process/container or host network setting changed. The later target
ran the single complete AT invocation; no failed AT predicate was retried.

## Requirement and AT coverage

UPD01/UPD02, §13.4 and §13.10: targeting, accepted current, actual offline
catch-up and new-node inheritance. UPD03/UPD05, §13.5–§13.9: verified artifacts,
state preservation, correlated final sample, journaled switching, five health
checks and physical rollback. D-098/D-132 preserve immutable evidence; D-110,
D-111, D-125 and D-127 remain bounded to their existing lab scenarios. These
historical lane labels do not redefine the wider original §19 cases.

| Case | Result | Deciding evidence |
| --- | --- | --- |
| AT23 | pass | Real root executable, D-053 challenge and D-063/64 traversal, health 5/5, local success, central succeeded/confirmed. |
| AT24 | pass | Actual 180 s offline window, zero-online snapshot, accepted baseline retained; catch-up plus a separate inherited-default job, no snapshot backfill. |
| AT25 | pass | HASH_MISMATCH before transaction/switch/health; candidate absent; last-good runs and failed receipt is accepted/confirmed. |
| AT26 | pass | Candidate tunnel failure → physical last-good restoration; restored health 5/5; central rolled_back / needs_attention=true / unconfirmed=false. |
| AT27 | pass | Same-run AT26 physical witness, disagreement withholding, agreement confirmation, bounded audit and stale ACK protection. |

The shared installer wrapper alone installs, with its exact TEST ONLY loopback
spec. Generated endpoint/origin, enrollment server CA and enrolled client paths
are read back unchanged; whole-config SHA-256 before/after agrees. No reporting
adapter or missing-result-url path. Systemd service observation is separate from
reporting. Server trust is distinct from the client issuer. Installer 27/27.
Route capture: **13 POSTs = 11 explicit stimuli + 1 automatic success + 1 automatic
stopped-producer failure**. Repeat flush stays 1 → 1; history=1/outbox=0 before
stimuli and local seal/ACK match central. Synthetic D-127 correction does not
supply the physical rollback witness.

F18-01 remains 1/1/2 apply/replay/recover, CHECKPOINT_FAILED, unchanged current
and seal, blocked forwarding. Positive D-125 barrier/boot/deadline and 0/0
watermarks correlate; no newly counted bytes are claimed. Uncached seed exits 0
with the original generated config and signed URL, adds only seeded_baseline,
and restores the original archive byte-identically. No private-mirror claim.

Every updater stop, in chronological fixture order (milliseconds):

| Stop (complete chronological list) | 2babf9b6 command duration | 2babf9b6 stage bracket ms | 2babf9b6 outcome | Current command ms | Current same-stage bracket ms | Current outcome / agent |
| --- | --- | ---: | --- | ---: | ---: | --- |
| installer switch | not recorded | not retained | success inferred from archived installer success | 8.499 | 22.741 | exit 0; exited |
| installer pre-migrate repeat | not recorded | not retained | success inferred from archived installer success | 2.986 | 15.519 | exit 0; already stopped |
| AT23 switch | not recorded | 442.504 | stop returned success | 443.556 | 448.428 | exit 0; unobserved |
| AT23 pre-migrate repeat | not recorded | 227.706 | stop returned success | 229.330 | 233.839 | exit 0; already stopped |
| AT26 failed-candidate switch | not recorded | 442.037 | stop returned success | 441.713 | 447.106 | exit 0; exited |
| AT26 pre-migrate repeat | not recorded | 229.594 | stop returned success | 227.499 | 232.093 | exit 0; already stopped |
| AT26 rollback | not recorded | 949.085 | rollback completed; Stop error is discarded by product | 435.179 | 933.306 | exit 0; exited |
| AT24 catch-up switch | not recorded | 438.58 | stop returned success | 439.400 | 445.742 | exit 0; exited |
| AT24 catch-up pre-migrate repeat | not recorded | 231.621 | stop returned success | 229.107 | 236.324 | exit 0; already stopped |
| AT24 inherited switch | not recorded | 437.058 | stop returned success | 437.477 | 445.587 | exit 0; exited |
| AT24 inherited pre-migrate repeat | not recorded | 232.947 | stop returned success | 225.727 | 231.965 | exit 0; already stopped |


Historical successful stops are inferred from advancement to stopped/migrated;
rollback's Stop return is discarded by product code, so its historical command
outcome cannot be proved from successful rollback alone. Every old process
termination/timeout observation is unknown. Full timestamps and raw current
wait status/lease fields are in `agent-stops.json`; no direct historical timing
is invented. Stage intervals include work other than Stop and current observer
readback overhead. Their small differences do not establish a shutdown regression.

The regular stop command uses 100 × 0.2 s polls plus scan work, and its supervisor
escalates after 25 × 0.2 s polls plus scan work; systemd reports 90 s. These are
separate from the agent's controlPassTimeout()+5 s (at most 10 s) local-work budget.
Every command returned 0; no command timeout was observed. Every stop readback
retains a lease row; all **11 start observations** also have a stored lease both
before and immediately after, with no read errors. Later renewal/expiry and
worst-case shutdown boundaries are not measured by those readbacks.

The product difference requested by the prompt is:

```text
$ git diff --stat 45209609d33d567c5f04d49441799cdd3efac59b..HEAD -- node packaging internal
 node/agent/engine.go                         |  23 +++-
 node/agent/followup21_netlab_test.go         |   4 +
 node/agent/followup31_netlab_test.go         | 123 ++++++++++++++++++
 node/agent/followup31_restart_test.go        | 115 ++++++++++++++++
 node/agent/followup31_test.go                | 100 ++++++++++++++
 node/agent/review_d056_test.go               |   4 +
 node/enforce/apply.go                        |  25 ++--
 node/enforce/apply_capture.go                |  29 +++++
 node/enforce/command_lifecycle.go            | 187 +++++++++++++++++++++++++++
 node/enforce/command_lifecycle_test.go       | 175 +++++++++++++++++++++++++
 node/enforce/command_resources_linux.go      |  97 ++++++++++++++
 node/enforce/command_resources_linux_test.go |  51 ++++++++
 node/enforce/command_resources_other.go      |   8 ++
 node/enforce/executor.go                     |   4 +-
 node/enforce/timing.go                       |   6 +
 node/internal/fakecontrol/usage_test.go      |  14 +-
 16 files changed, 946 insertions(+), 19 deletions(-)
```

#369 supplies the safe boundary and bounded local completion after cancellation;
control I/O remains cancellable. #365 supplies opt-in TEST ONLY enforcement
lifecycle observation. The regular config has TestMode=true but no timing path;
the installer config has TestMode=false and no timing path. The hook is not
enabled here. Packaging/internal are unchanged from the Follow-up 12 product
baseline. This lane modifies none of the product files.

## Files

- `history/follow-up-12/`: 44 unchanged root files, archive note and SHA256SUMS.
- `history/follow-up-12-stop-journal/`: separately pinned four-field readback of
  the retained baseline journals, never a rewritten old archive.
- `history/follow-up-13-startup/`: immutable failed startup before any AT began.
- Owned stop observer, container-only installation hook, collection script and
  kernel-status test; archive test; provenance/current follow-up labels.
- `mk/at-updates.mk`: configurable port slot, still only the two required targets.
- Fresh AT, installer, transport, origin, derived evidence, source/run metadata,
  complete stop table/JSON, restart-lease readbacks, current report/requests,
  procedure, historical forwarding links and D-132 audit.
- `.agent/REPORT.md`, `.agent/last.md` and logs are ignored, never committed.
