# Updates follow-up 14 — 2026-11-01

Run `2704600f-e076-4a59-9338-09befa2af70e`: **regular AT23–AT27 PASS 5/5**, installer **27/27**,
shared-wrapper install **exit 0**, **13 result POSTs**, physical AT26 rollback,
and uncached-fallback probe. One regular invocation, live target exit 0,
Go test 514.704 s. Product main `f680d6ddb8fd19979c0471a2ce06ffc6e16c7cd2`;
measured harness `64893bb3de8e81ea3ca13e07871ca0cdc2455b85`. No product sources changed by this lane.

All **11 stop commands return 0**. Six distinct agent terminations have their
own parent evidence of **exit 0**; five repeated stops retain the same prior
receipt and count no new exit. Every stop stays below 15 s, no SIGKILL attempt
or actual signal termination is observed, and the stored lease is present at
all 11 stop and 11 start readbacks. Stop findings: **0**. Requests: **0**.

D-098 archive commit `b9924151` came first: all **48** follow-up 13 root files
of run 759d244a were copied byte-identically and pinned under
[history/follow-up-13](history/follow-up-13/ARCHIVE.md). All 478 earlier history
files, including follow-up-13-startup, remain unchanged. Plain linked files
remain present. The [D-132 audit](d132-audit.md) records four unchanged product
reference sites and immutable-only evidence pins.

## Requirement and AT coverage

UPD01/UPD02 (§13.4, §13.10) cover target selection, accepted current, natural
offline catch-up and new-node inheritance. UPD03/UPD05 (§13.5–§13.9) cover
verified artifacts, retained state, correlated final sample, journaled switch,
five health checks and physical rollback. The historical lane AT labels are
unchanged; they do not replace the broader original §19 populations.

| Case | Result | Deciding evidence |
| --- | --- | --- |
| [AT23](AT23.md) | pass | Real candidate, D-053 challenge and D-063/64 traversal; health 5/5; local success and central succeeded/confirmed. |
| [AT24](AT24.md) | pass | Natural 180 s offline window and zero-online snapshot; accepted baseline retained; catch-up and separate inherited-default job; original snapshot unchanged. |
| [AT25](AT25.md) | pass | HASH_MISMATCH before transaction/switch/health; candidate absent; last-good running; failed receipt accepted and confirmed. |
| [AT26](AT26.md) | pass | Candidate tunnel_probe fails; physical last-good restored with health 5/5; central rolled_back, needs_attention=true, unconfirmed=false. |
| [AT27](AT27.md) | pass | Same-run physical rollback witness, disagreement withholding, agreement confirmation, bounded audits and stale ACK protection. |

The shared installer wrapper alone installs using its exact TEST ONLY loopback
opt-in. Generated result URL, exact origin, enrollment server CA and enrolled
client paths are consumed unchanged, with identical whole-config SHA-256
readback. No reporting adapter is added. The 27/27 checks include automatic
result delivery, local ACK/seal matching and immutable reporting configuration.
Server trust remains distinct from client issuer trust.

[Route capture](installer-delivery-transport.json): 13 POSTs = 11 explicit
contract stimuli + 1 automatic success + 1 automatic stopped-producer failure;
10 responses are 202 and 3 are 409. Repeat flush stays 1 → 1 and the local
pre-stimulus counts are history=1 / outbox=0. Synthetic correction is separate
from AT26's physical rollback witness.

[Uncached fallback](artifact-origin.json) exits 0 with the unchanged generated
config and original signed URL. Withholding only the verified archive adds
only seeded_baseline; the archive is restored byte-identically. No private
mirror deployment is claimed. F18-01 keeps apply/replay/recover exits 1/1/2,
CHECKPOINT_FAILED, unchanged current/seal and blocked forwarding. Quiet 0/0
watermarks are not newly counted traffic.

## Every updater stop

| Stop (complete chronological list) | 759d244a command ms / exit | 759d244a agent | Current command ms / exit | Current agent |
| --- | ---: | --- | ---: | --- |
| installer switch | 8.499 / 0 | exited 0 | 52.688 / 0 | exit 0 / signal null |
| installer pre-migrate repeat | 2.986 / 0 | already stopped | 3.028 / 0 | exit 0 / signal null (same receipt; no new exit) |
| AT23 switch | 443.556 / 0 | unobserved | 448.124 / 0 | exit 0 / signal null |
| AT23 pre-migrate repeat | 229.330 / 0 | already stopped | 231.254 / 0 | exit 0 / signal null (same receipt; no new exit) |
| AT26 failed-candidate switch | 441.713 / 0 | exited 0 | 449.261 / 0 | exit 0 / signal null |
| AT26 pre-migrate repeat | 227.499 / 0 | already stopped | 227.978 / 0 | exit 0 / signal null (same receipt; no new exit) |
| AT26 rollback | 435.179 / 0 | exited 0 | 433.314 / 0 | exit 0 / signal null |
| AT24 catch-up switch | 439.400 / 0 | exited 0 | 439.985 / 0 | exit 0 / signal null |
| AT24 catch-up pre-migrate repeat | 229.107 / 0 | already stopped | 226.091 / 0 | exit 0 / signal null (same receipt; no new exit) |
| AT24 inherited switch | 437.477 / 0 | exited 0 | 434.169 / 0 | exit 0 / signal null |
| AT24 inherited pre-migrate repeat | 225.727 / 0 | already stopped | 222.130 / 0 | exit 0 / signal null (same receipt; no new exit) |

[Full identity and receipt table](agent-stops.md) and [machine-readable rows](agent-stops.json)
retain every pre-stop launch ID, PID/start ticks, command duration/outcome,
parent exit status and UTC observation time. The regular fixture follows the
exact devstack 25 read sequence: retain current ID before stop; read that ID's
receipt after return; read it again after starts even when current ID changes.
All retained regular receipts remain identical after restarts. The first AT23
exit is now observed as a new sample, not a reconstruction of run 759d244a.

The installer parent is systemd. Its native waitid code=1/status=0 and parent
exit timestamp are exported separately; the first stop also captured raw kernel
wait status 0. The encoded wait(2) equivalent is labeled derived, never presented
as a devstack receipt. The repeat has no new kernel sample or new exit. Systemctl
shows a one-second timestamp, while devstack reports RFC3339Nano; neither is an
exact kernel exit instant. A container's exit status is never used as an agent
exit status. The installer retains its generated service and reporting setup.

The collector uses docker cp, which also works after a whole-container stop;
the current updater stops are service-only. Devstack 26's existing supervisor
waits 15 s after TERM, PID1 forwards TERM/INT, and compose/clone grace is 25 s.
The matching sigkill sidecar represents attempted escalation only; the parent
wait receipt determines the outcome. This lane changed no shutdown deadline.

The installer first stop increased from 8.499 to 52.688 ms; all observations
remain within budget with the AT predicates intact. This is a measurement,
not evidence of a shutdown regression. No worst-case bound is claimed. A future
stop reaching 15 s, SIGKILL, command failure or lost stored lease must remain a
failed row; agent failures belong to agent-core and stop handling to updater.
[Restart lease readbacks](restart-leases.json) cover immediate persistence only.

## Product baseline

```text
$ git diff --stat 5bd751026073444087b32ac2a7115f8566104f8b..HEAD -- node packaging internal
 internal/enrollment/revoke.go                      |   6 +
 internal/nodes/generation_hold.go                  | 136 ++++
 internal/nodes/generation_hold_integration_test.go | 422 ++++++++++
 internal/nodes/generation_hold_store.go            | 181 +++++
 internal/nodes/generation_hold_test.go             | 111 +++
 .../nodes/httpwire/composition_integration_test.go |   2 +-
 internal/nodes/observations.go                     |   7 +
 internal/nodes/publication_evidence.go             |  22 +-
 internal/operations/operations.go                  |  33 +
 node/agent/decide.go                               |   6 +
 node/agent/engine.go                               |  75 +-
 node/agent/followup21_test.go                      |  12 +
 node/agent/followup32_test.go                      | 252 ++++++
 node/agent/followup33_intersection_spec_test.go    | 890 +++++++++++++++++++++
 node/agent/freeze.go                               |   2 +-
 node/agent/local_budget.go                         |  48 ++
 node/agent/memberpath.go                           |   2 +-
 node/agent/observe.go                              |  45 +-
 node/agent/probeexc.go                             |   2 +-
 node/agent/quarantine.go                           |  13 +
 node/agent/review_r18_generation_test.go           | 761 ++++++++++++++++++
 node/agent/serving.go                              |  23 +-
 node/agent/stale_hold.go                           |  60 ++
 node/agent/sync.go                                 |   2 +-
 node/agent/usage.go                                |   4 +-
 node/enforce/command_lifecycle.go                  |  23 +-
 node/enforce/command_lifecycle_test.go             |  59 +-
 node/enforce/command_resources_linux.go            |  23 +-
 node/enforce/command_resources_linux_test.go       |  36 +-
 node/enforce/command_resources_other.go            |   9 +-
 node/enforce/followup23_listing_test.go            |  92 +++
 node/enforce/parse_nft.go                          |  19 +-
 node/enforce/review_r18_lifecycle_test.go          | 164 ++++
 packaging/installer/followup9_test.go              | 165 ++++
 packaging/installer/install.go                     |   8 +-
 packaging/installer/services.go                    |  19 +
 36 files changed, 3672 insertions(+), 62 deletions(-)

```

These are upstream product differences since follow-up 13, not edits by this
lane. The measured source commit retains exactly starting main's product bytes.

## Registers and remaining measurements

F11-01 remains CLOSED (shared wrapper installs and original reporting works).
F11-02 remains CLOSED (#351 retained-archive versus guarded fallback contract).
F18-01 remains the intended stopped-producer negative. **Updates F13-01 remains
recorded for run 759d244a**: its first AT23 wait status is unobserved. Devstack
25/26 delivered the observation support, and this new sample consumes it; no
historical row or archive was rewritten. Members' F13-01 is unrelated.

Updater 19's gaps remain: physical arm64, real power loss/filesystem durability,
complete disk/migration fault populations, fleet rollout/mixed populations and
three join sources, production central, worst-case 30 s wait / 100 ms poll,
near-deadline publication, malformed/oversized/special-file or foreign /
alternate-spelling / other-boot checkpoints, withdrawal after sampling,
mid-handshake crashes, original-deadline preservation, identical republication
and after-deadline publication. Distinct-endpoint CA deployment, private mirror,
real-device kernel/throughput and contention causes remain unmeasured.
Cross-compilation is not physical arm64 evidence. This quiet run proves no
worst-case shutdown duration or long-term lease behavior.

## Execution and checks

Admission: zero other suites before startup and after startup; **wait 0 s**.
Port slot 54000; foreground only; worktree disk scratch; GOCACHE read through
go env. All owned Docker objects are removed after the run. No sudo, host
network change, shared Postgres change, foreign-container operation or external
test traffic. No retry for a pass and no relaxed predicate.

Required lane lint/live/unit, arm64 build, handover, root lint, vet, diff check
and committed-evidence hygiene checks pass; amd64 build and full go test ./...
also pass. Root lint uses go-with-disk-tmp= solely to avoid its shell cleanup
trap; worktree TMPDIR/GOTMPDIR remain set and all lint checks are retained.
See the worktree Follow-up 14 hand-off for real output tails. No push, PR or tag.

### Requests to orchestrator

2026-11-01 — **0 requests**. All required current agent outcomes are observed.
Historical missing observations and the unmeasured product scenarios above
remain explicit limitations; no outside-owned file was changed.
