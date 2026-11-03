# F27-01: deadline-guard hand-over design

2026-11-03 · follow-up 31 · design only · base main
`cc0ed4ae713f6c8242171f923bdc799a0c07a74c`.

**Recommend the retained-deadline intersection guard (C1) for design review,
not implementation approval. Keep the current guard in production.** C1 can
avoid the demonstrated drops for unchanged members while preserving their old
kernel deadlines and applying revocations in the candidate transaction. It has
an explicit trade: after a late commit or death before budget approval, those
members can continue under old, independently expiring authority. Today's guard
closes **all** members at that cut. None of the stock-kernel candidates below
preserves that exact closure guarantee while being hitless at the same cut.
Only the owner may accept that trade; without that decision the recommendation
is to retain A. This is not a blanket established-flow exception.

Read in this order:

1. [Current guarantees, code citations, crash cuts and measured cost](current-guarantees.md).
2. [Three candidates, exact transaction boundaries and recommendation](candidates.md).
3. [Proof obligations, failure injections and netlab plan](proof-obligations.md).
4. [Predeclared D-133 A,B,A,B comparison and decision rule](d133-comparison.md).

The starting interpretation is the [follow-up 30 roadmap row](../../ROADMAP.md)
and [PR #418's orchestrator reading](https://github.com/TED-H-com/Freedom-VPN/pull/418),
read before design work, plus the prior [proposal](../G2-followup30-evidence/guard-outcome.md#proposal-only--separate-d-133-work)
and [drop ledger](../G2-followup30-evidence/drop-ledger.md). Seven exact guard drops
demonstrate a mechanism. **B1 remains FAIL and unexplained.** Both follow-up 30
live runs remain host-unqualified; follow-up 28 gate 2 and every other historical
failure retain their status. This follow-up runs no live diagnostic or sketch
and uses no reservation. Source/document review settles the design alternatives
without claiming any new kernel behavior has been measured.

The no-weakening rule is a prerequisite, not a performance criterion that a
clean comparison can override. D-120, its 1.5-second timeout, D-134's 100 ms
reserve/install budget and 1 ms truncation, and D-135's baseline-without-old-policy-
restoration are unchanged. The staged design's outstanding closure decision and
cross-lane reader/writer compatibility must be settled before B is implemented.

### Requests to orchestrator

1. **2026-11-03 — owner decision, `docs/ROADMAP.md` and the decision register in
   the orchestrator's private repository (outside this lane):** review C1's
   G5/G6 trade: all-member immediate closure after the guarded commit versus
   continued forwarding only for the proven old/new intersection, bounded by
   unchanged old expiry. A SIGSTOP while holding the lock can defer watchdog
   closure until that expiry. No implicit approval or new production bound is
   assumed. If the old closure behavior is mandatory, reject hitless C1 and
   retain the current guard. Record B1 and host qualification unchanged.
2. **2026-11-03 — agent-core, `node/agent/plane.go`, `engine.go`, `restart.go`,
   `serving.go`, `decide.go`, and `node/watchdog/watch.go`:** if C1 is approved,
   provide trusted prior-model/boot/verified-state provenance under the existing
   lock, classify restrictions before additional I/O, preserve failure withdrawal,
   ACK and meter bindings, reject interrupted hand-overs at adoption, and upgrade
   every baseline writer to the bank/token-aware format before activation.
   Review the extra control-loop occupancy bound and same-boot/signal cases.
   Tests in `tests/netlab/agent/*` and agent-owned files remain that lane's work.
3. **2026-11-03 — updater/installer and acceptance owners, `node/updater/*`,
   `packaging/*`, `tests/e2e/*`, `tests/handover/*`:** inventory independent
   enforcement readers, baseline writers and old-release rollback compatibility
   before changing the four-chain / `allowed_v4` representation. Assign native
   revocation, metering and physical arm64 acceptance. No partial rollout with an
   old writer is permitted by this proposal.
4. **2026-11-03 — orchestrator scheduling and review:** after design approval and
   implementation proofs, schedule the quiet D-133 slot with no other lane
   session, freeze A/B and the common observer, and review the fixed rule in
   [the comparison declaration](d133-comparison.md). Current launch admission
   improvements answer follow-up 30 request 2; no admission-script edit or new
   diagnostic is requested here. The new TEST ONLY trace adapter must represent
   a restrictive guard truthfully, without changing the D-120 analyzer.

Only those cross-lane implementation/decision items stop here. The complete
design and test/comparison plan are delivered for review. All edits are confined
to this new documentation directory; ignored hand-off files carry the gate
receipts. Follow-ups **25–30 remain byte-identical (D-098)**, including all 1,818
payload files, six manifests and both forms of compressed evidence. The final
[verification receipt](verification.json) records the documentation checks.
