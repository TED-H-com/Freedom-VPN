# Updates follow-up 10 — 2026-10-31

Run `695f977c-b587-4768-88f1-0514f3a4f1ac`, harness `0df8599dc84d4b85e48182f20aa4b0fc741bc72a`, product main `fb5e002549f79e8ef61bebcf42cce1678ab65cb1`. The installer extension passes all 23 checks. **The live make target exits 2** (Go test 1): regular AT23 cannot start its candidate, and AT24–AT27 remain blocked. Only this run supplies current pass claims.

## AT23–AT27

| Case | Result | Deciding observation |
|---|---|---|
| [AT23](AT23.json) | blocked | Candidate logs `agent: active release identity is inconsistent`; root executable is absent while `bin/fvpn-agent` exists. Journal ends `recovery_required` / `ROLLBACK_FAILED`, last-good empty; no health challenge/attestation. |
| [AT24](AT24.json) | blocked | No verified last-good after AT23; later scenario not executed. Its required checks remain absent, so the live gate fails. |
| [AT25](AT25.json) | blocked | No verified last-good after AT23; later scenario not executed. Its required checks remain absent, so the live gate fails. |
| [AT26](AT26.json) | blocked | No verified last-good after AT23; later scenario not executed. Its required checks remain absent, so the live gate fails. |
| [AT27](AT27.json) | blocked | No verified last-good after AT23; later scenario not executed. Its required checks remain absent, so the live gate fails. |

The shared regular release builder places the binary under `bin/`; the agent's active-release validator checks the root-level binary. This explains the observed process refusal by static source inspection, with runtime paths and log captured in AT23. Fixing the shared builder/service manager is outside this lane. See F10-01; no product validation was weakened.

## Installer → updater → central

| Command / stimulus | Status / exit | Journal / local observation | Central view |
|---|---|---|---|
| Wrapper `install --json` | installed / 0 | Product agent active; pinned NetBird active on `/run/fvpn-netbird/netbird.sock` | Before update, current/last-good both null |
| `seed` default umask; exact replay under 077 | 0 / 0 | Same verified sequence-10 seed, tree hash and timestamp | No promotion from seed alone |
| Fresh updater oneshot and `run` | 0 / 0 | No desired state or pending work; no guard created | No update summary |
| `upgrade --json` sequence 11 | staged / 0 | Pending record exists; current stays sequence 10 | No staged-as-completed claim |
| `apply --pending` with real agent | succeeded / 0 | Byte-equal D-125 latch/checkpoint; epoch finalized; current → sequence 11, units restart, five health checks pass; sealed receipt `source=installer` | Accepted 202; source installer, batch null, current=last-good=sequence 11; D-110 job recomputed |
| Same-key and new-key exact receipt replay | 202 / 202 | Original local seal hash verified against accepted body | Stored ACK; ETag remains "8" |
| Modified receipt; wrong job ID | 409 / 409 | HTTP contract stimuli, no local journal mutation | `ACK_CONFLICT`; `STATE_CONFLICT` reason `job_id_mismatch`; no pointer/version change |
| Stop producer, withdraw real checkpoint; stage/apply sequence 12 | staged 0; apply 1 | `CHECKPOINT_FAILED` at `final_sample`; null checkpoint; forwarding disabled; sequence 11 retained | Failed evidence accepted 202; current=last-good remains sequence 11 |
| Restore exact pending bytes; sealed apply replay; recover | 1 / 2 | Same receipt/hash, current unchanged, forwarding latch retained | No duplicate receipt |
| D-127 / r16 M-2 later rollback correction | 202 | **TEST ONLY HTTP stimulus**, derived from real success; no physical rollback claimed | Current=last-good restored to sequence 10; state rolled_back; ETag "9" → "10" |
| Replay first and correction; third changed outcome | 202 / 202 / 409 | Two outcomes for success job, first receipt retained | Both stored ACKs; third `ACK_CONFLICT`; correction ETag stable |
| Stage sequence 13, tamper archive, apply | 2 / HASH_MISMATCH | Refused before switch; local current stays sequence 11 | No candidate promotion claimed |
| `upgrade --json` sequence 10 | 1 / DOWNGRADE | No pending record written | No candidate promotion |
| Agent/seed/job/lock boot identity | equal | All match kernel `97c018d7-a925-4723-9127-83c097de1c8a` | Same boot in sealed result |

Sequence IDs: 10 `01a19291-8468-7ac6-b7d1-52517e8b73f1`, 11 `01a19291-8f30-7b66-9c35-c7deb94d063f`, 12 `01a19291-aa81-77b3-bedb-97ec115f2a72`, 13 `01a19291-b9c9-7558-8304-ab86a78cf8d0`. Successful job `40b192cc-48c1-5648-9d5f-1f007842ca62`; result `4f1d4378-a66b-5256-bc05-b582e2647ce1`. Failure result `455a30da-a0fd-5151-9946-905ede8c6654`. Complete commands and responses are in [installer-handoff.json](installer-handoff.json).

D-125: `barrier_id` `4557daef-fdee-4f8a-a37f-79725452a3c0` equals persisted wait identity `4557daef-fdee-4f8a-a37f-79725452a3c0` byte-for-byte. The checkpoint names epoch `01a19291-fb18-7639-9565-b10df45bfdc6`; durable watermark `0` and ACK `0` match the sealed result. The durable epoch is closed with reason `final_sample`. This fixture is quiet and unpublished; it does not remeasure withdrawal of active member traffic.

Serving withdrawal and nonzero accounting are consumed from [devstack 19, run d6aaa798](../../runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-journaled-upgrade.md) and [D-102 responses](../../RESPONSES.md#updater--installer--2026-10-31-follow-up-19-measured-result), as requested. The stopped-producer negative here withdraws genuine checkpoint evidence; it does not synthesize an invalid checkpoint or test disappearance after a checkpoint has already been accepted.

The generated updater config still has empty `result_url`. Central-delivery checks explicitly supply the owned endpoint, CA and allowed origins; installer-configured reporting is **not** claimed (F10-02). D-127/r16 M-2 uses a labeled HTTP correction; regular AT26's physical rollback is blocked in this run.

## Owner rows consumed

[source-main.json](source-main.json) pins the roadmap rows: installer #279/#302/#311/#326; updater #307/#310/#323; agent-core #320 (D-125); fleet-update #300/#322 (D-110/D-111/D-127); devstack #284/#332. F9-01/02/03/05/06 are remeasured successfully in the installer extension. F9-04 central acceptance is resolved; its separate generated-reporting configuration issue remains F10-02.

## Evidence and limits

D-098 was committed first as `22c1f775`: [65 byte-identical files and SHA-256 pins](history/follow-up-9/ARCHIVE.md), including the first-run directory as it was. Diagnostic attempts [one](history/follow-up-10-first-run/README.md), [two](history/follow-up-10-second-run/README.md), and [three](history/follow-up-10-third-run/README.md) remain pinned; they supply no current pass claims.

Physical arm64 execution ([existing operator row](at23-arm64.json)), three join sources, mixed populations, and the full disk/migration fault matrix remain open. This lab offers amd64 execution; real amd64 and arm64 binaries were cross-built for the signed artifacts.

Run instructions are in [RUN.md](RUN.md). Current requests are [F10-01 and F10-02](requests.md); no shared or product file was edited.
