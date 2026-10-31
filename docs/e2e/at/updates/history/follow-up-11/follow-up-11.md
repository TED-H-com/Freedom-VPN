# Updates follow-up 11 — 2026-10-31

Run `daddad24-b6f7-4c32-ba8e-1c7d24f76ea2`, measured harness `3ac13f7e115edf22e2ec9d3a6450733ec1fb7b6a`, product main `96ec0630e4834e13a3a14d65553ba8108bb5da55`.
**All five scoped regular rows pass, the installer extension passes 27/27,
and the live make target exits 0** (Go test 511.968 s). Only this run supplies
current pass claims. These historical lane scenario labels do not redefine
complete §19 acceptance.

## AT23–AT27

| Case | Result | Deciding observation |
| --- | --- | --- |
| [AT23](AT23.json) | pass | Exact release-root executable is running; real D-053 challenge and D-063/D-064 probe traversal, five passing checks, journal `locally_succeeded`, current=last-good=1.0.1; central succeeded and confirmed. |
| [AT24](AT24.json) | pass | Actual 180 s offline window expires; target snapshot has zero online nodes and preserves accepted 1.0.1. Real catch-up reaches 1.0.2; a later new node completes its own inherited-default job, without backfilling the original snapshot. |
| [AT25](AT25.json) | pass | Tampered archive produces `HASH_MISMATCH` before transaction/switch/health; root candidate executable absent, last-good still running; null-health failed receipt accepted and confirmed. |
| [AT26](AT26.json) | pass | Candidate's real tunnel probe fails; updater physically restores last-good and verifies five restored checks. A current ACK yields central `rolled_back`, `needs_attention=true`, `unconfirmed=false`; conflicts retain the original receipt. |
| [AT27](AT27.json) | pass | Mandatory `check_physical_rollback=true` names the same-run AT26 journal/batch/current/health. Disagreeing observations withhold confirmation; agreeing observations confirm; repeated unexpected-release audit is bounded; stale ACK cannot replace the newer job. |

The physical rollback transaction is `01a19330-0749-7ce3-aad0-ddf926cada16`.
Candidate version `0.1.0` actually reports `tunnel_probe=failed`; restored
version `1.0.1` reports all five checks passed. The exact restored executable
is `/opt/vpn-node/releases/01a1932d-6558-70cc-97ba-064e3ba3b7b2/fvpn-agent`.
AT27's witness is this journaled physical switch/restart/health and accepted
central result. It does not borrow the installer correction stimulus.

## Installer → updater → real central

| Command / stimulus | Status / exit | Journal / local observation | Central view |
| --- | --- | --- | --- |
| Shared wrapper `install --json` | usage / 64 | Refuses loopback without explicit test opt-in; identity, usage and current absent before/after | No update receipt |
| Same spec via `ProbeInstallerExit`, plus TEST ONLY `--allow-loopback` | installed / 0 | Real agent and NetBird active; generated reporting points to the shared authenticated local proxy | Current/last-good initially null |
| `seed` default umask; exact replay under 077 | 0 / 0 | Same verified sequence-10 baseline/hash/timestamp | Seed alone does not promote current/last-good |
| Fresh updater oneshot and `run` | 0 / 0 | No desired/pending work; forwarding guard absent | No update summary |
| `upgrade --json` sequence 11 | staged / 0 | Verified pending record; current stays sequence 10 | Staged is not complete |
| `apply --pending`, real producer | succeeded / 0 | Correlated latch/checkpoint, real final-sample epoch, 10→11 switch, unit restarts, health 5/5; immutable seal | Automatic 202 before manual HTTP stimuli; `source=installer`, batch null, current=last-good=11; recomputed D-110 job |
| Repeat product `flush` with generated config | 0 | Outbox empty, exactly one ACK history entry and matching seal; config hash unchanged | Same ACK/admin view; no additional POST in route capture |
| Same-key and new-key exact receipt replay | 202 / 202 | Explicit HTTP contract stimuli, separate from automatic delivery | Stored ACK, ETag stays `"8"` |
| Modified receipt; wrong job ID | 409 / 409 | No new local apply | `ACK_CONFLICT`; `STATE_CONFLICT/job_id_mismatch`; pointers/version unchanged |
| Stop producer, withdraw real checkpoint; stage/apply sequence 12 | staged 0; apply 1 | `failed/final_sample/CHECKPOINT_FAILED`, null usage checkpoint, current 11 retained, forwarding blocked | Automatic failed evidence accepted 202; current=last-good=11 |
| Exact pending replay; `recover` | 1 / 2 | Same failed seal/hash; no switch; latch retained | No duplicate product receipt; explicit failed-receipt replay accepted |
| D-127/r16 M-2 correction | 202 | **TEST ONLY HTTP stimulus**; local current stays 11; this row is not physical rollback | Current=last-good=10; ETag `"9"`→`"10"`; original receipt retained |
| First/correction replay; third changed outcome | 202 / 202 / 409 | Two immutable outcomes for successful job | Stored ACKs, stable correction ETag; third `ACK_CONFLICT` |
| Stage 13; tamper retained artifact; apply | 2 / `HASH_MISMATCH` | Refused before switch; current 11 retained | No candidate promotion |
| `upgrade --json` sequence 10 | 1 / `DOWNGRADE` | No pending record written | No candidate promotion |
| Post-extension uncached `seed` of current 11 | 0 | Retired installer fixture only; same generated policy; downloaded from unchanged release origin; cache restored; only `seeded_baseline` added | No apply/result submission or central promotion claimed |


The [generated config](installer-handoff.json) has result URL
`https://127.0.0.1:9444/api/v1/agent/update-results`, origin exception exactly
`https://127.0.0.1:9444`, server trust `/etc/vpn-node/ca.pem`, and the enrolled
client cert/key paths. This is the shared installer's split-listener enrollment
proxy, which verifies the current client and forwards to the real central at
`https://agent.example.invalid:9443`; no reporting URL/CA/origin is inserted by
this lane. The server CA matches enrollment trust and differs from the client
issuer. Whole-config hashes match before/after; `result_url_missing` is absent
from captured updater commands and service logs.

The automatic success seal is result `c14779ae-bf73-5a58-ae3c-3236ffb604f2`,
job `16498646-7504-5a4b-bb93-dec222ed1924`, payload hash
`3e704a4c9cd227099d392f2d0640d36c30090fa7179930650ff4a5f959002cd9`.
The locally stored ACK equals the real central ACK received at
`2026-10-31T18:17:13.343497Z`; the hash matches the immutable central payload.
The admin `GET /api/v1/admin/nodes/{id}` exposes source installer and no batch.
[Central route logs](installer-delivery-transport.json) contain 13 requests
through the extension: 11 explicit curl contract stimuli, one automatic success,
and one automatic stopped-producer failure. Repeated product flush adds none.
This separates exactly-once automatic transmission from deliberate HTTP replays.

D-125 checkpoint `barrier_id=ecb40e92-439e-4a59-8ab4-c8faa743394a` equals the
persisted latch identity within the original boot/deadline. The quiet checkpoint
names epoch `01a1932d-ffe5-7801-8451-9f6648b6527c`, durable/ACK watermarks 0/0,
with durable `close_reason=final_sample`; no new usage bytes are claimed.
Agent-core 28's [restart/meter record](../../../node/followup28-iso-f6-03-2026-10-31.md)
was read before evaluating these transitions. Its ordinary-stop open-epoch
behavior does not remove the updater's final-sample requirement. No new restart
or checkpoint defect was observed. Serving withdrawal/nonzero accounting remain
attributed to devstack 19 rather than remeasured by this quiet extension.

F18-01 remains the intended stopped-producer negative: apply/replay/recover
**1/1/2**, failed seal retained, forwarding blocked. Neither this row nor the
archived old stopped-producer failure is reported as a successful upgrade.

## Artifact-origin decision

The predicted rejection **did not occur**. Initial install and upgrade retain
verified archives, so normal hand-off uses those files. A separate
[same-run uncached seed probe](artifact-origin.json), after fixture retirement,
also exits **0** with generated config unchanged. The shared release hostname
resolves to an owned non-private/non-CGNAT Docker origin. The existing
`allowed_artifact_origins` field exempts private/CGNAT origins; it does not deny
all other hosts. This is the D-046 behavior documented by
[node-updater.md](../../../updates/node-updater.md) and implemented in
`node/updater/verify.go`. Separate immutable HTTPS URLs are documented by
[release-process.md](../../../updates/release-process.md).

Neither control-origin hosting nor a generated policy override was introduced.
The probe restores the cached archive byte-identically and retains current,
last-good and transaction outcome; it adds only a verified baseline for current.
It supplies no private-mirror deployment claim. F11-02 names the stale offline
installation paragraph rather than inventing an origin-policy defect.

## Owner evidence, archive and scope

[Source/roadmap pins](source-main.json) include #340 fleet-update 13, #341
installer 7, #342 updater 18, #343 devstack 20 and #344 agent-core 28.
Devstack 20's [regular run d6b01926](../../runs/2026-10-31/followup20/d6b01926-de6f-47e4-803b-ae3ea5c603df/update-good.md)
is consumed as its **11/0/0** owner result with the
[D-102 answer](../../RESPONSES.md#updates--2026-10-31-follow-up-20-f10-01-d-102).
Its measurement was not rerun; this lane ran its own complete requested sequence.

D-098/D-132 was the first commit, `52bba4ad`: all **39** living-root records of
`695f977c-b587-4768-88f1-0514f3a4f1ac` are byte-identically
[archived with SHA-256 pins](history/follow-up-10/ARCHIVE.md).
The three earlier F10 diagnostic archives are untouched. The
[D-132 audit](d132-audit.md) confirms no product test reads living AT evidence.

Physical arm64 execution ([unchanged operator row](at23-arm64.json)), three
join sources, mixed populations, and the full disk/migration fault matrix
remain open. This lab executed amd64; arm64 cross-build is not hardware proof.

[Run instructions](RUN.md) describe the foreground run, unchanged generated
config, separate transport/origin captures and cleanup. Two nonblocking dated
[requests](requests.md) remain: devstack's explicit test loopback input and the
updater guide's stale retained-archive description. No outside-owned file,
product code, contract, dependency or shared Postgres setting changed.
