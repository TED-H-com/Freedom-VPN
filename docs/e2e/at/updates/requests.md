# Updates requests — follow-up 12, 2026-10-31

Run `2babf9b6-86a3-489d-a678-0bd447f2f8b0`, product main
`45209609d33d567c5f04d49441799cdd3efac59b`, measured harness `5b2ae074`.
Follow-up 11's complete request record and measurement are preserved in
[history/follow-up-11](history/follow-up-11/ARCHIVE.md).

### Requests to orchestrator

2026-10-31: **No new requests.** Request count: **0**.

### Owner closures

| Item | Disposition | Evidence |
| --- | --- | --- |
| F11-01 | **CLOSED** | Devstack 21 / #352 added shared `installerSpec.AllowLoopback`, gated with the exact local `ControlURL`. This lane's actual `Stack.RerunInstaller` returns **installed / exit 0**, with all 27 installer checks passing. See `shared_installer_spec`, `install_arguments_from_shared_spec`, `install_path` and `install_wrapper` in [installer-handoff.json](installer-handoff.json). No install fallback is invoked. The [D-102 answer](../../RESPONSES.md#updates--2026-10-31-follow-up-21-f11-01-d-102) is consumed. |
| F11-02 | **CLOSED by #351** | [Updater 19](../../../updates/node-followup19-2026-10-31.md#f11-02-retained-archive-and-missing-cache-fallback) corrects initial/same-release retained verified archives versus absent-cache guarded downloads. [Installer 8 / #353](../../../node/install-followup8-2026-10-31.md) reconciles the producer fields and documentation. No product or outside-owned edit is needed here. |
| F18-01 | Intended D-125 negative retained | Stop the real producer and withdraw its checkpoint: apply/replay/recover **1/1/2**, sealed `CHECKPOINT_FAILED`, unchanged current and failure seal, blocked forwarding. This is never a successful upgrade. The old failed campaign stays archived. See the [D-102 response](../../RESPONSES.md#updater--2026-10-31-follow-up-21-f18-01-d-102). |

Updater 19 / #351 accepts updates 11's generated reporting and physical rollback;
installer 8 / #353 pins the same reporting fields independently. Their tests
read no living AT evidence. All 43 former root files are now available under
immutable history paths before any follow-up 12 publication (D-098/D-132).
Agent-core 29 / #356 adds opt-in TEST ONLY host context around Apply without
changing production ordering or deadlines. This follow-up is the first regular AT23–AT27
campaign on that tree; no install-contention cause or worst-case bound is
claimed.

No new artifact-origin input is requested. The [uncached probe](artifact-origin.json)
uses the original generated policy and signed artifact URL; the lab's owned
non-private origin needs no private/CGNAT exception. Distinct-endpoint CA and
private-mirror deployments remain unmeasured. The full remaining measurement
list is in [follow-up 12](follow-up-12.md#remaining-measurements).
