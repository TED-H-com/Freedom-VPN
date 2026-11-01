# Follow-up 13 — shutdown regression, 2026-11-01

Run `759d244a-2dc4-460c-9f82-e37ce0ac7560`: regular **AT23–AT27 5/5**,
installer **27/27**, shared-wrapper **installed / exit 0**.
Product main `5bd751026073444087b32ac2a7115f8566104f8b`, measured harness `20d1f6e83d4a0dce97e2cf87e4ccd4a31dc580a5`;
live make exit **0**, Go test **517.66 s**.
[Run metadata](run.json), [source and roadmap rows](source-main.json),
[procedure](RUN.md), [all stop comparisons](agent-stops.md).

D-098 came first: commit `6f1eb10c` copied all **44** Follow-up 12 root files
byte-identically to [history/follow-up-12](history/follow-up-12/ARCHIVE.md),
with the same manifest shape as follow-up 11. Every source blob was compared.
The failed first attempt and all earlier archives remain untouched. The
[D-132 audit](d132-audit.md) retains history-only evidence pins.

## Regular AT predicates

| Case | Result | Deciding observation |
| --- | --- | --- |
| [AT23](AT23.json) | pass | Actual release-root executable, real D-053 challenge / D-063–64 traversal, health 5/5, local success and central confirmation. |
| [AT24](AT24.json) | pass | Real 180 s offline window, zero-online snapshot, accepted baseline preserved; catch-up and later node's distinct inherited-default job complete without backfilling the snapshot. |
| [AT25](AT25.json) | pass | Tampered archive rejected with HASH_MISMATCH before transaction, switch or health; failed receipt accepted/confirmed, last-good remains running. |
| [AT26](AT26.json) | pass | Genuine candidate tunnel failure → physical last-good restoration, health 5/5, central rolled_back / needs_attention=true / unconfirmed=false. |
| [AT27](AT27.json) | pass | Mandatory same-run physical AT26 witness, mismatch withholding and agreeing confirmation, bounded audit and stale ACK protection. |

AT26 and AT27 name physical rollback transaction `01a1967e-4a4f-74fb-82d1-452ec3c4a454`.
The installer HTTP correction is a separate synthetic D-127 stimulus.
UPD01/UPD02 (§13.4, §13.10) cover targeting/catch-up; UPD03/UPD05 (§13.5–§13.9)
cover verification, state preservation, correlated final sampling and rollback.
These existing lane labels do not redefine or close the complete original §19
architecture/source/fault population.

## Installer and reporting

The shared `Stack.RerunInstaller` alone installs the fixture. Its TEST ONLY
loopback opt-in and exact local ControlURL are read back from the shared spec.
Generated result URL, exact origin, enrollment server CA, client certificate/key
paths and whole-config hash remain unchanged. Server trust differs from the
client issuer, and `result_url_missing` is never observed. The service-command
observer is isolated from reporting; no URL, CA or origin adapter is introduced.
See [installer-handoff.json](installer-handoff.json).

[Route capture](installer-delivery-transport.json): **13 result POSTs**
= 11 explicit HTTP stimuli + one automatic success + one automatic failure.
Repeat-flush counts stay **1 → 1**, independently of central deduplication.
Before explicit stimuli, history=1/outbox=0 and the seal/ACK match central;
source=installer, batch=null, current=last-good=sequence 11. The genuine success
receipt is `bb589670-4dbc-5750-98a8-41c516c54db3`; independently recomputed D-110 job
`abd0e3da-bfdc-5631-b82a-ed69336fdcb2`. Replays and bounded correction preserve immutable
payload/ACK history; modified, wrong-job and third outcomes are rejected.

F18-01 remains the intended D-125 stopped-producer negative: apply/replay/recover
**1/1/2**, sealed CHECKPOINT_FAILED, unchanged current/failure seal and blocked
forwarding. The positive checkpoint/wait share the original barrier, boot and
deadline. Quiet durable/ACK watermarks 0/0 do not claim new counted bytes.

The [uncached probe](artifact-origin.json) withholds the retained archive and
seeds using the unchanged generated config and signed URL: exit 0, no origin
rejection or added permission. Only seeded_baseline changes; the cached archive
is restored byte-identically. This owned non-private/non-CGNAT origin does not
prove private-mirror or distinct-endpoint CA deployments.

## Every updater stop, compared

[The full table](agent-stops.md) lists **11 current stops**, including
switches, repeated pre-migration stops, failed-candidate switch and physical
rollback. Direct current monotonic command durations sit beside all available
same-stage timestamps from run 2babf9b6. Its nine regular stage brackets were
read from retained journals before reset, separately archived with SHA-256 pins.
The old run has **no direct command durations or process wait statuses**; its
installer journal was not retained. Those missing measurements cannot be
recreated retrospectively and are not substituted with stage durations.

All current stop commands returned 0. No stop command timeout, SIGKILL or lost
stored lease was observed; [restart readbacks](restart-leases.json) retain a
lease row before/after every updater start. The current wait-status sampler missed
**1** reaped process(es): node stop 1 (final_sampled). These rows explicitly say
**unobserved**, so graceful exit is not claimed for every agent. This is
**F13-01**, a fixture-observation gap, not a demonstrated agent/updater failure.
The AT predicates stand on their own evidence; the full termination measurement
remains incomplete. No retry was performed to turn that row into a pass.

The regular fixture's public stop uses 100 × 0.2 s polls plus scan work; its
supervisor escalates after 25 × 0.2 s polls plus scan work. Systemd reports a
90 s stop timeout. These are separate from agent-core 31's local-work budget
of controlPassTimeout()+5 s (at most 10 s). No worst-case bound or scheduling
cause follows from these measurements. Netlab 22's enforcement lifecycle hook
remains opt-in and is not enabled by this lane's configs.

## Baseline and admission

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

Only node paths differ from Follow-up 12's product baseline; packaging/internal
are unchanged. This lane changes no product path. #369 adds safe-boundary
shutdown and bounded local completion while control I/O remains cancellable;
#365 supplies the opt-in TEST ONLY enforcement observation.

One [startup failure](history/follow-up-13-startup/startup.json) remains exit 2:
port 42443 was occupied by an unrelated host listener, before any AT/installer
test started. No host process was changed. The original note proposed waiting;
the final decision used the shared harness's free 54000 slot. Both admission
checks found one other suite. One Go AT invocation then supplied every current
predicate; no failed AT was retried or relaxed. Docker names remain exclusively
`fvpn-w4-updates-*`; the harness tears down the stack.

## Registers and remaining measurements

F11-01 stays **CLOSED** by the successful shared installer path. F11-02 stays
**CLOSED by #351**, with installer 8's aligned archive documentation. F18-01
stays the intended negative. [Requests](requests.md): **1**.

Updater 19's gaps remain: physical arm64; actual power loss/filesystem durability;
complete disk/migration fault matrices; fleet rollout, mixed populations and
three join sources; production central; worst-case 30 s wait / 100 ms poll and
controlled near-deadline publication; malformed/oversized/special-file,
foreign/alternate-spelling identity and other-boot checkpoints; withdrawal after
sampling; mid-handshake crash, original-deadline preservation, identical
republication and after-deadline publication. Quiet 0/0 is not new accounting
continuity. Cross-builds are not arm64 hardware evidence. Real-device kernel,
throughput and contention-cause/availability bounds remain unclaimed.
