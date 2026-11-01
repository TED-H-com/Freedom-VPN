# Follow-up 12 — shared installer wrapper, 2026-10-31

Final run `2babf9b6-86a3-489d-a678-0bd447f2f8b0` passes the scoped regular **AT23–AT27 5/5** and
installer **27/27** on product main `45209609d33d567c5f04d49441799cdd3efac59b`. Measured harness
`5b2ae0748bbbffb0d370d4842bb0623e74e844b7`; live make/test exit **0**, Go test **505.765 s**.
The shared `Stack.RerunInstaller` is the sole installation path and returns
**installed / exit 0**. No `ProbeInstallerExit` install fallback or generated
reporting adapter is used. See [run metadata](run.json), [source record](source-main.json)
and [run procedure](RUN.md).

D-098/D-132 came first: commit `e9b8cf5a` archives **all 43** former root files
for run `daddad24-b6f7-4c32-ba8e-1c7d24f76ea2`, compared byte-for-byte with their
main blobs, under [history/follow-up-11](history/follow-up-11/ARCHIVE.md).
Earlier archives remain unchanged. New evidence-file digest pins name only
immutable `history/` paths; product tests do not read living AT evidence.

## Installer and transport

The [shared spec readback](installer-handoff.json) has `allow_loopback=true`
and exact `control_url=https://127.0.0.1:9444`. Devstack 21's `installArgs`
gates the TEST ONLY flag on both fields. Production default remains false.
`install_arguments_from_shared_spec` is a diagnostic reconstruction from the
shared source/spec; `install_wrapper` is the actual shared-wrapper outcome.
The lab SNAT in that diagnostic is a TEST ONLY Docker address, not deployment
configuration. `ProbeInstallerExit` remains only the existing upgrade-command
exit diagnostic; it does not install this node.

Generated reporting is consumed unchanged: result endpoint
`https://127.0.0.1:9444/api/v1/agent/update-results`, exact origin exception,
enrollment server CA `/etc/vpn-node/ca.pem`, and enrolled certificate/key paths
under `/var/lib/vpn-node/identity/`. The server CA matches enrollment trust and
differs from the client issuer. Whole-file SHA-256 readbacks match before/after;
`result_url_missing` is never observed. The authenticated local proxy forwards
to real central code in the test deployment; this is not production-central
acceptance.

[Route capture](installer-delivery-transport.json) records **13 POSTs**:
**11 explicit HTTP stimuli + one automatic success + one automatic failure**.
The repeat-flush route counts are **1 before / 1 after**, independently of
central deduplication. Before manual stimuli, local history is 1/outbox 0,
its seal and ACK match central, and central reports `source=installer`, batch
null, current=last-good=sequence 11. Success result
`649be9b7-51b5-56ee-b87f-fdefbf8fee47` has independently recomputed D-110 job
`9eaf0900-be9c-55e8-a0af-743e975745c7`; ACK received at `2026-10-31T22:08:11.799943Z`.
Replay returns 202; modified, wrong-job and third-outcome submissions return 409.
The D-127 correction stays an explicitly synthetic HTTP stimulus.

D-125's genuine producer and persisted wait share barrier
`1ac07276-d8d1-47da-94f5-d9b8052eb023` within the original boot/deadline.
The quiet final-sample epoch has durable/ACK watermarks **0/0**; no new counted
bytes are claimed. Stopped-producer apply/replay/recover remain **1/1/2** with
`CHECKPOINT_FAILED`, unchanged failed seal/current and blocked forwarding.
F18-01 is the intended negative, never an upgrade success.

The [uncached probe](artifact-origin.json) resolves the current pointer,
withholds only its cached archive, and runs seed with the original signed URL
and generated config: **exit 0**, no origin-policy rejection and no new origin
input. The owned release origin is non-private/non-CGNAT. Only a verified
`seeded_baseline` is added; current/last-good, failed outcome and config remain
unchanged. The original archive is restored byte-identically.

## Regular cases and requirement scope

| Case | Result | Deciding observation |
| --- | --- | --- |
| [AT23](AT23.json) | pass | Actual release-root agent, real D-053 challenge and D-063/D-064 traversal; health 5/5; local success and central confirmation. |
| [AT24](AT24.json) | pass | Actual 180 s offline window, zero online nodes in snapshot, preserved accepted baseline; 1.0.2 catch-up and later node's separate inherited-default job, original snapshot unchanged. |
| [AT25](AT25.json) | pass | Bad archive gives `HASH_MISMATCH` before switch/health; candidate absent, last-good running; failed null-health receipt accepted and confirmed. |
| [AT26](AT26.json) | pass | Candidate tunnel probe fails; physical switch back to last-good, restored health 5/5; current ACK and central `rolled_back`, `needs_attention=true`, `unconfirmed=false`. |
| [AT27](AT27.json) | pass | Mandatory same-run physical AT26 witness; mismatches withhold confirmation, agreeing observations confirm, bounded audit and stale ACK cannot replace newer job. |

The AT26 journal and AT27 witness name physical transaction
`01a19403-7f30-7790-a987-751842a77782`, independently
of the installer correction. UPD01/UPD02 (§13.4, §13.10) cover targeting and
catch-up; UPD03/UPD05 (§13.5–§13.9) cover verified state-preserving switching,
checkpoint refusal and rollback. These historical lane labels retain their
scoped meaning; they do not close the whole original §19 population/fault matrix.

## Consumed closures and baseline

- Updater 19 / **#351** accepts updates 11, adds
  `TestUPD05_InstallerReportingConfigShape`, and closes **F11-02** by correcting
  retained-archive versus absent-cache documentation.
- Devstack 21 / **#352** supplies the exact-local-origin opt-in and D-102 answer.
  **F11-01 is CLOSED by this lane's shared-wrapper exit 0**, independently of
  devstack's earlier 33b9621a endpoint measurement.
- Installer 8 / **#353** reconciles producer/consumer fields and adds
  `TestUPD05_RenderedReportingConfigShape`; no field divergence remains.
- Agent-core 29 / **#356** adds opt-in TEST ONLY host samples around Apply and
  sanitized command status, without changing production ordering/deadlines.
  Follow-up 12 is the first regular AT23–AT27 campaign on this tree. Relative
  to follow-up 11's `96ec0630`, `node/` and `packaging/` differ (8 files,
  736 insertions, 2 deletions); `internal/` does not. No product file is changed
  by this lane. Host contention is not established as a cause by this run.

## Retained first attempt

Two invocations occurred. The [first attempt](history/follow-up-12-first-attempt/ARCHIVE.md),
`1f2b57de`, had regular 5/5 and installer 27/27 but **make exit 2**: this lane's
new uncached probe passed a relative symlink as `seed --tree`, rejected before
download with `CHECKPOINT_FAILED`. Its 17 records and incorrect unconditional
narrative are preserved with an explicit erratum; no historical failure is
relabelled. The owned fix resolves the pointer and makes the narrative
conditional. The final run repeats every case and supplies every current pass;
no row is borrowed from the failed attempt. No product or hosting workaround
was introduced.

## Remaining measurements

Updater 19's remaining gaps stay open: physical arm64; actual power loss and
filesystem durability; complete disk/migration fault matrices; real fleet
rollout, mixed populations and three join sources; production central;
worst-case **30 s wait / 100 ms poll** bounds and controlled near-deadline
publication; malformed/oversized/special-file checkpoints, foreign or
alternate-spelling UUIDs, other-boot evidence and withdrawal after sampling;
mid-handshake crashes, original-deadline preservation, identical republication
and publication after deadline. Quiet 0/0 here does not replace new nonzero
accounting-continuity measurement. Distinct-endpoint CA/private-mirror
deployments, real-device kernel/throughput and contention-cause/availability
bounds remain unmeasured. Cross-building arm64 is not physical execution.

[Requests to orchestrator](requests.md): **0**. F11-01 and F11-02 are closed;
F18-01 remains the intended stopped-producer negative.
