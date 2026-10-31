# Enrollment acceptance — follow-up 7

Run `59ef4bf8-7d4a-490f-a74d-6b54394375e0`, **2026-10-31T08:58:59Z–2026-10-31T09:05:27Z**.
Source `5412e257ee0735877239ae18543f6b81a98a2ae2`, clean at invocation, based on merged main through #294.
All fixtures and identifiers are TEST ONLY.

| AT | Status | Pass / blocked / fail | Evidence |
|---|---|---|---|
| AT06 | pass | **424 / 0 / 0** | [JSON](AT06.json), [rows](AT06.md) |
| AT07 | blocked | **36 / 5 / 0** | [JSON](AT07.json), [rows](AT07.md) |

AT06 preserves all 376 follow-up-6 checks and adds 48 evidence-file scans:
24 new archive/attempt files, scanned in both phases. Both phases record
**0 HTTP 429**. All historical failure reviews are resolved, with the original
failed observations retained.

ENR-02/03 now starts from the ordinary enrolled, serving node: a matched and
authorized native member, one signed lease member, six live member accept rules,
a fresh serving heartbeat and accepted forwarding report. The clone conflict
withdraws those rules and produces an identity_frozen heartbeat within **227 ms**
of the central conflict transaction, a conservative bound inside the measured
**5,000 ms heartbeat**. A replay of the real report returns identity_frozen=true.
The durable SQLite latch survives an agent process restart; a fresh heartbeat
and accepted report remain non-serving, with zero member rules and zero lease
members. This is independent evidence, not promotion of isolation's empty-grant
proof. It is process restart evidence, not OS reboot or physical arm64 evidence.

Seven installer subset checks pass on the #294 exports: quiescent rerun hashes,
rerun identity, controlled-reinstall identity and retained files, non-serving
hold/release with exact ledger snapshots, and service restoration after all
three invocations. Install/rerun/reinstall return installed/already_current/
reenrolled. Reinstall retains the Node, creates a new incarnation and retires
the old identity in mode 0700. Its certificate receives 403 IDENTITY_REVOKED on
both desired-state and observations. Hold returns retryable 503, desired-state
stays 200, release reaches central validation (422 for the malformed probe),
and the three snapshots are byte-identical. **UnACKed usage is zero**.

The five composed rows remain blocked:

| Rows | Blocker / proof |
|---|---|
| installer_rerun, controlled_reinstall, counted_unacked_usage | D-116: installer owner marker causes HOOK_PRIORITY_MISMATCH, preventing serving. [Devstack proof](../../runs/2026-10-31/followup17/a7c41436-82b3-4665-b75c-e2887f2e1009/installer-owner-chain.md). |
| identity_ledger_preserved, historical_usage_retained | D-105 / D-112: revision 14 defines retired-stream delivery, but central acceptance and node upload are not merged on this source. Before-plus-held-byte equality and historical attribution remain unproved. D-116 also prevents nonzero installer usage. [Contract](../../../contracts/README.md), [lane requests](requests.md). |

The [enrollment answer](../../RESPONSES.md#enrollment--2026-10-31) is present;
missing devstack exports are no longer the blocker. No container state was
hand-edited to repair product behavior, and no outside-owned pin changed.

Follow-up 6 was archived byte-identically **before** new records were written:
[checksums and original records](history/follow-up-6/README.md).
The initial diagnostic attempt has an explicit [lookup-failure review](history/d83ac47d-0074-42d7-bb3b-65a2db536be5/REVIEW.md).
Its unrun join phase is not promoted; the corrected run completes both phases.

Both live probe processes and all teardowns exit 0. The live make target exits
2 because of the five documented blockers. Area lint, root Go lint, vet,
untagged Go tests and evidence hygiene pass. Cleanup is **0 containers /
0 networks / 0 volumes / 0 images**. See [runbook](RUN.md) and [requests](requests.md).
