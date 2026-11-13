# Handover follow-up 52 — guide statement corrections

This cut records **one merged pull request: #527**, through
`6972be03e4742a13c3e02e4c612ece37fd314fb8`. The cutoff is the roadmap commit after
#527, whose squash is `d37bf5a085610bd54597b5e3d09f306ab9a763a9`.
[Follow-up 51](follow-up-51.md) covered #524–#526 and was itself merged as #527;
it therefore belongs in this range, just as the preceding cut belonged in its range.

The [source manifest](../../tests/handover/testdata/followup52-main.json) records
the title, full merge and head commits, merge time and roadmap commit. The
summary below describes the [merged pull request](https://github.com/TED-H-com/Freedom-VPN/pull/527).
This cut does not re-verify that delivery or claim a new run, measurement,
deployment, restore or CI result.

## Merged delivery

| PR | Merged title | Merge commit | Head commit | Effect and boundary |
|---|---|---|---|---|
| [#527](https://github.com/TED-H-com/Freedom-VPN/pull/527) | handover: follow-up 51 — cut #524–#526, and correct the deployment, calibration, restore and backup guides against the code | `d37bf5a085610bd54597b5e3d09f306ab9a763a9` | `60a6e81025c469756221230b899e98acc5aadac5` | Publishes follow-up 51 and corrects deployment, calibration, restore and backup guidance, with two targeted operator-guide corrections. Operators get statements tied to the implementation and explicit carried gaps; developers get frozen prior inputs and bounded edit checks, without a new deployment or acceptance claim. |

## Guide revisions at this cut

These are corrections to statements at the cutoff, with no new procedure.
The [edit manifest](../../tests/handover/testdata/followup52-edits.json) records
the exact old and new text and its source evidence.

| Guide | Correction and boundary |
|---|---|
| [Deployment](deploy.md) | Enrollment returns `node_incarnation_id`; the serving list is `minimum_serving_version.serving_release_ids`. Two dedicated resolvers remain the specification's recommended starting point for resilience (§12.2); the code accepts one resolver, which has no redundancy. See the [API contract](../../contracts/openapi.yaml), [release-support schema](../../contracts/schemas/policies/release-support-payload.schema.json), [node configuration](../../internal/config/nodes.go) and [resolver applier](../../node/dns/applier.go). |
| [Deployment](deploy.md) | At the conntrack retry bound or with `conntrack` missing after preflight, the agent stops serving and attempts the baseline. Forwarding reasons include `safety_check_failed`; `ISOLATION_POLICY_INVALID` with the conntrack detail requires the baseline to be in force, otherwise `internet_only` reports `ISOLATION_BASELINE_MISSING`, meaning isolation is not confirmed. A retained Apply failure withholds generation acknowledgement. See [agent enforcement](../../node/agent/engine.go) and [agent checks](../../node/agent/decide.go). |
| [Calibration](calibration.md) | Fair-use status comes through `GetMemberStatus` / `GetAdminMemberStatus`, with the enforce-mode ack-reader check, and the usage query carries `api.FairUseStatus`. Both guides identify `anonymousLim`, `classify`, the inline request middleware, `walkChecked`, `checkLargeJSON` and `identityFieldMatches`. Bodies up to `canonjson.MaxBytes` pass `canonjson.Canonicalize`; only on the agent mTLS listener do those bodies also pass `walkChecked` with an identity callback. Larger bodies use `checkLargeJSON`, with streamable schema checks in the same walk. See the [FUP engine](../../internal/fup/engine.go), [usage query](../../internal/usage/queries.go), [middleware](../../internal/httpapi/middleware.go) and [JSON walker](../../internal/httpapi/jsondoc.go). |
| [Restore drill](restore-drill.md) | Step 3.2 keeps its restricted-recovery-startup gap notice once. The gap and the recovery ordering remain unchanged. |
| [Operator guide](operator-guide.md) | Corrects `meter_path`, token-file selection, the executor call, loopback evidence and datapath fields, `addrID` revocation identity, and combined `verifyServing` read-back. The conntrack bound counts durably reserved serving-Apply delete batches, including slots consumed before deletion; it is not a count of consecutive failed commands. See [token intake](../node/install.md#token-intake-182-enr03), [enforcement](../../node/enforce/apply.go) and [revocation accounting](../../node/agent/revocation.go). |
| [Operator guide](operator-guide.md) | Updater space accounting adds the signed reserve without a 512 MiB floor; check and rollback bounds count transaction-step entries, with no configurable retry intervals. The fair-use read-back is the owning member's usage response; a member with no published carrying exit reports `applying: false` regardless of a `policy.activate` operation's state; configuration defaults use 1 MiB browser/agent/webhook and 8 MiB usage caps. See [updater space accounting](../../node/updater/engine.go), [transaction steps](../../node/updater/apply.go), [fair-use status](../../internal/fup/engine.go) and [configuration defaults](../../internal/config/config.go). |
| [Backups](../ops/backups.md) | The verify unit's prebuilt migrator is linux/amd64. Without `--migrate-bin`, [verify.sh](../../deploy/backup/verify.sh) builds that architecture. |
| [Calibration](calibration.md) and [operator guide](operator-guide.md) | Fair-use ACKs use the latest stored observation without checking its age or current node control status. An acknowledged exit becoming offline or stale does not by itself make `applying` true again; false does not establish current liveness. The activation payload uses `type`, evaluation runs as `fup.evaluate`, and the policies worker passes fair-use activation and member-generation events directly to `Compiler.RecompileBatch`. Plan activation runs a batch-limited sweep but has no direct compile call; the plan-only rate-change gap remains when no generation event triggers a compile. See the [ACK query](../../internal/fup/engine.go), [staleness sweep](../../internal/nodes/observations.go) and [policies worker](../../apps/worker/jobs_policies.go). |
| [Operator guide](operator-guide.md) | Revoke-step eligibility, pending status and completion follow `advanceNodeOperationsOnObservation`. Admin pending-delete diagnostics are `pending`, `ISOLATION_POLICY_INVALID` and `member_forwarding_unconfirmed`; reasons and resolutions are in audit events. One clean counting observation can complete the step while another fresh exit without a pending isolation error has not applied its generation. See [observation processing](../../internal/nodes/observations.go) and [public operation projection](../../internal/operations/operations.go). |
| [Operator guide](operator-guide.md) | Replaces stale incarnation, error, snapshot-test, renewal, revocation, grace, discovery, probe and schema names with current code references. Concurrency defaults to 128 and eviction scans the LRU tail. `toRecovery` supplies blank rollback health only with no supplied or stored health at `api.UpdateStageRollback`; four checks fail and `identity_and_state` comes from `checkPersistent`, which can pass. See [configuration](../../internal/config/config.go), [limiter](../../internal/httpapi/middleware.go), [recovery](../../node/updater/apply.go) and [blank health](../../node/updater/attest.go). |
| [Operator guide](operator-guide.md) | Provisional-grant waiting has the healthy description only when otherwise healthy; independent enforcement/probe failures need investigation. Protective-fence audits record success and the actual fence fields; peer-denial and identity-rejection transition tracking are separate; peer denials use device, then member, then fleet fallback resources. Omitting `error_code` gets no result receipt; an otherwise valid, job-bound failed result with `error_code: null` gets a rejection receipt. Both rejections preserve the prior projection, and acceptance distinguishes job advancement from current-observation updates. Revocation retention uses two address filters, stale-observation grace has one outstanding mark with an applied-generation floor, the public-key digest is hex-encoded without another hash, and baseline attempts are distinguished from confirmed isolation. Retained Apply/dependency failures withhold generation acknowledgement; meter-path failures have their own cause and forwarding reason and alone do not withhold it. See [agent decisions](../../node/agent/decide.go), [fence audits](../../internal/devices/fence.go), [denial paths](../../internal/devices/audit.go), [update results](../../internal/fleetupdate/results.go), [revocations](../../node/agent/revocation.go), [grace](../../internal/nodes/observations.go), [incarnations](../../internal/nodes/incarnations.go) and [the plane interface](../../node/agent/plane.go). |
| [Calibration](calibration.md) and [operator guide](operator-guide.md) | The 60-second central grant-less expiry cap applies only to published, registered nodes that are not ready or have frozen mappings; other grant-less leases keep ordinary expiry. The agent's planned renewal spacing can be shortened by the renewal lead and lengthened by `Retry-After`. See [central lease issuance](../../internal/leases/leases.go) and [agent renewal](../../node/agent/sync.go). |
| [Operator guide](operator-guide.md) | Retained Apply/dependency failures still withhold acknowledgement; eligible live generation mismatches can be reported and repaired at the same generation with the current lease, without a baseline on successful repair. After datapath validation, quarantine attempts its marker, member-link shutdown and baseline independently. A procedure error can coexist with confirmed containment; evidence distinguishes the first and latest wrapper calls from Apply’s own quarantine. See [serving verification and repair](../../node/agent/serving.go), [quarantine](../../node/enforce/quarantine.go) and [reported containment evidence](../../node/agent/quarantine.go). |
| [Calibration](calibration.md), [operator guide](operator-guide.md) and [member guide](member-guide.md) | The calibration provisional report is qualified by otherwise healthy state and independent failure codes. Calibration and operator guidance describe planned renewal spacing. The member guide describes eligible renewal attempts and links to the operator entry for provisional reports and renewal spacing; dated records remain unchanged. See [agent decisions](../../node/agent/decide.go) and [renewal scheduling](../../node/agent/sync.go). |

The [Chinese](../DELIVERY.md#交接指南修訂) and
[English](../DELIVERY.en.md#handover-guide-revisions) delivery guides record
these revisions alongside the revisions of follow-ups 50 and 51. Latest-cut
entry points advance to this cut; references to follow-up 51's content remain.

## Dated-record erratum

The [2026-10-30 restore record](restore-drill.md#25-measured-on-the-lab--live-at30-second-half-221-2026-10-30-run-0f5fb60d-f6a7-45c6-8ebe-175e0a480bd8-docse2eatsecopsat30md-docse2eatsecopsrestore-reconcilemd)
names `UploadOnce`. The method is `Uploader.Once` in the
[usage uploader](../../node/internal/usageupload/upload.go). The dated record
stays as written; this name correction adds no recovery evidence.

## Carried gaps and verification boundary

Carried gaps remain exactly as [follow-up 51 recorded them](follow-up-51.md#existing-procedures-linked-and-carried-gaps).
The [delivery guide's verification limits](../DELIVERY.en.md#verification-scope-and-known-limits) and the dated
acceptance records retain their original scope. This cut supplies no missing
procedure and closes no product or acceptance gap.

The [pre-cut provenance](../../tests/handover/testdata/followup52-snapshots.json)
pins exact copies from `6972be03`. The claims check permits 91 named edits across
14 source documents, preserves headings and historical passages, resolves
relative links and heading anchors, and keeps earlier assertions against frozen
inputs. [Prior assertion provenance](../../tests/handover/testdata/followup52-prior-assertions.json)
records the read-path-only changes; earlier claims checks still inspect the
live test source. These are documentation consistency checks, not live-system evidence.
