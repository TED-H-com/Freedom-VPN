# Handover follow-up 53 — guide statement corrections

This cut records **five merged pull requests, in merge order: #529, #528, #530, #531 and #532**, through
`e055143ead570c9ff4357a910d0969c854077b72`. The cutoff is the roadmap commit after
#532, whose squash is `74d7ea2c908b712474c45d467e203a1c39f02d2b`.
[Follow-up 52](follow-up-52.md) covered #527 and was itself merged as #529,
so #529 belongs in this range.

The [source manifest](../../tests/handover/testdata/followup53-main.json) records
full titles, merge and head commits, merge times and roadmap commits. The linked
merged pull requests are the sources for these delivery summaries. This cut does
not re-verify them or claim a new run, measurement, deployment, restore or CI result.

## Merged deliveries

| PR | Merged title | Merge commit | Head commit | Effect and boundary |
|---|---|---|---|---|
| [#529](https://github.com/TED-H-com/Freedom-VPN/pull/529) | handover: follow-up 52 — cut #527, and correct the deployment, calibration, restore-drill, operator, member and backup guides against the code | `dd0a898421e5299bf3e62ad5edd1b2caca64e64b` | `27f54b2a2ba09963de0c57e7310f289ee1d7ab0c` | Publishes follow-up 52, correcting deployment, calibration, restore, operator, member and backup statements. Its frozen inputs and bounded edits preserve earlier claims without new deployment or acceptance evidence. |
| [#528](https://github.com/TED-H-com/Freedom-VPN/pull/528) | backup: isolated restores start from configuration they write themselves, keep only the restore's recovery settings and clean up after a failure; script guards no longer depend on the locale | `8a04d92bbe343dd9abb46a05e2862f089a8b6841` | `40cc040b98726bdb96a0df53340efe76773b3125` | Isolated restores use independent configuration and retain only their own validated recovery settings; failure cleanup respects resource ownership. Script identifier guards are independent of the caller’s locale, with fixed drill inventory and locale controls. |
| [#530](https://github.com/TED-H-com/Freedom-VPN/pull/530) | worker: run the node staleness sweep as a scheduled job, so a node that stops reporting becomes stale and then offline instead of staying online | `e126f45b98d3fd9c0913f106cba35b9fb40046a1` | `8309c2d2dec94bd75de3b1d50229b3c3932bf988` | The worker schedules receipt-age sweeps so silent nodes become stale and offline. This status classification does not itself withdraw routes or prove local forwarding has stopped. |
| [#531](https://github.com/TED-H-com/Freedom-VPN/pull/531) | portal: the admin node summary and overview state only what each node's fields show, so stale, offline, enrolling and expired nodes are not called online or serving | `c5d2a489157cbad19ebb57211589139604cc38b3` | `24af2fa3121765db71440c203aa6a2472da836bb` | Admin summaries use the node fields and ordered rules, with a green list badge only for online_serving. The overview compares report times as instants and displays the selected API string or the no-report label. |
| [#532](https://github.com/TED-H-com/Freedom-VPN/pull/532) | nodes: complete a member or device operation's nodes step only when every exit it affects has applied the change or is recorded as unconfirmed | `74d7ea2c908b712474c45d467e203a1c39f02d2b` | `4c36007fc89d5b9d18be720e494de3ec943eac29` | Member/device operations finish the nodes step from a completed capture and per-exit pinned outcomes, including explicitly unconfirmed exits. Migration 00196, the compile fence and transaction-bound reads support that model; per-target detail has no API/UI surface. |

## Guide revisions at this cut

The [edit manifest](../../tests/handover/testdata/followup53-edits.json) records
the exact replacements and source evidence. These are statements at the cutoff,
not new procedures.

| Guide | Current statement and boundary |
|---|---|
| [Deployment](deploy.md), [calibration](calibration.md) and [operator guide](operator-guide.md#11-daily-operator-tasks) | Central receipt-age status uses the configured online window and scheduled sweep; local heartbeat/watchdog timing remains separate. Renewal lead is configured and clamped, with populated-lease jitter. Publication still requires its operation and route read-back. |
| [Operator guide](operator-guide.md#33-publishing-disabling-and-unpublishing-exits-put-apiv1adminnodesnode_idpublication) | Corrects verifier encoding, forwarding field names, retry/repair eligibility, provisional central expiry and device-list rate classification. Operation completion uses captured per-node pins and acknowledged/unconfirmed outcomes; an empty capture is evidence. |
| [Database index](11-database-and-retention.md#compile-and-capture-transaction-constraints) | Names `CompileFenceNamespace` and `CompileFencePurpose`, keeps reads/writes on the fenced transaction, and records lock order, capture evidence and migration 00196 before the new worker/control binaries. Existing evidence and retention guards remain. |
| [Restore drill](restore-drill.md#phase-2-restore-postgresql-17-and-freeze-usage-ingestion) and [backups](../ops/backups.md#21-postgresql-17-wal-archiving-fragment-deploybackuppostgresfvpn-archivingconf) | The isolated verifier uses independent configuration and narrow cipher inputs; the host-primary example names its database user. Restore worker inventories include the staleness sweep; drill inventory, identifier grammar and static-scanner limits are explicit. |
| [Member guide](member-guide.md) | Chinese wording now preserves the same historical follow-up 7 acceptance qualification as English, including AT30’s later pass; measured figures remain unchanged. |

The [Chinese](../DELIVERY.md#交接指南修訂) and
[English](../DELIVERY.en.md#handover-guide-revisions) delivery guides record
these revisions alongside follow-ups 50, 51 and 52. Latest-cut entry points
advance to follow-up 53; references to follow-up 52’s content remain.

## Dated-record errata

These five records stay unchanged. Each entry names the dated passage and links
to living guidance that states the current rule.

- `docs/handover/operator-guide.md`, [“Operator changes at #356”](operator-guide.md#operator-changes-at-356), including the 2026-11-03 rollback confirmation: baseline installation/read-back can fail; without a verified baseline isolation is unconfirmed. The verified conntrack-delete exception can keep serving below its durably reserved batch bound. The [living enforcement explanation](operator-guide.md#33-publishing-disabling-and-unpublishing-exits-put-apiv1adminnodesnode_idpublication) states those conditions.

- `docs/handover/operator-guide.md`, “3.4 Node Egress Probe & Sanitized Check Details”, [2026-11-01 — Node diagnostics (#386)](operator-guide.md#34-node-egress-probe--sanitized-check-details-d-058a-d-058b-171-nodeagentprobego-nodeagentprobeexcgo): failure stops serving and attempts a baseline; a failed install/read-back does not confirm isolation. See the [living enforcement explanation](operator-guide.md#33-publishing-disabling-and-unpublishing-exits-put-apiv1adminnodesnode_idpublication).

- `docs/handover/member-guide.md`, [“2026-10-31 更新 / update (D-087, #250)”](member-guide.md#2026-10-31-更新--update-d-087-250): heartbeat/doubling spacing describes planned slots. Renewal lead can shorten them; `Retry-After`, request I/O and ongoing work can delay actual attempts. See [current renewal scheduling](operator-guide.md#33-publishing-disabling-and-unpublishing-exits-put-apiv1adminnodesnode_idpublication).

- `docs/handover/README.md`, [“Follow-up 49 decisions register (2026-11-05 EST)”](README.md#follow-up-49-decisions-register-2026-11-05-est), expiry/rollback rows: baseline installation/read-back is fallible, and failed isolation is not proof that deny rules were installed. The verified conntrack-delete exception can retain serving below its reserved batch bound; see [current enforcement conditions](operator-guide.md#33-publishing-disabling-and-unpublishing-exits-put-apiv1adminnodesnode_idpublication).

- `docs/contracts/README.md`, [“Revision 13”](../contracts/README.md#revision-13) and [“Revision 11” / “D-071 `ErrorCode.AWAITING_MEMBER_GRANTS`”](../contracts/README.md#d-071-errorcodeawaiting_member_grants): the first identity-freeze renewal uses an explicit attempt/retry gate reset, not a same-generation Apply guarantee. Grant-less heartbeat/doubling slots remain subject to renewal lead, `Retry-After` and ongoing I/O/work; see [current retry gates and scheduling](operator-guide.md#33-publishing-disabling-and-unpublishing-exits-put-apiv1adminnodesnode_idpublication).

## Carried gaps and verification boundary

Carried gaps remain exactly as [follow-up 52 recorded them](follow-up-52.md#carried-gaps-and-verification-boundary).
The [delivery guide’s verification limits](../DELIVERY.en.md#verification-scope-and-known-limits)
and dated acceptance records retain their scope. This cut supplies no missing
procedure and closes no product or acceptance gap.

The [pre-cut provenance](../../tests/handover/testdata/followup53-snapshots.json)
pins exact source copies from `e055143e`. The claims check bounds every guide edit,
preserves existing headings and the [dated passages](../../tests/handover/testdata/followup53-historical.json),
and resolves relative links and heading anchors. [Prior assertion provenance](../../tests/handover/testdata/followup53-prior-assertions.json)
records read-path-only changes; earlier claims checks continue to inspect the
live test source. These are documentation consistency checks, not live-system evidence.
