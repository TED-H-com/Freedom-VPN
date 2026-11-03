# Isolation follow-up 17 requests — 2026-11-03

Three rows: two carried operator assignments and one owner capability item.
Run `0ec14b39-467e-4a37-a239-1a33ef12eb71`; [evidence](summary.md).

1. **Carried — AT11 platform/operator assignment.** Full native member-platform
   route/capture matrix stays blocked. No renewed ask.
2. **Carried — AT13 acceptance/operator assignment.** Full continuous crash,
   NetBird restart, kernel reboot, updater and physical-arm64 matrix stays blocked.
   Six exit-0 receipts do not close it. No renewed ask.
3. **New — w3-wire-nodes / w2-agent-core, F17-01: pure `stale_generation`.**
   The 16 accepted holds classify **12 check_failed / 4 isolation_unconfirmed**; pure classification
   remains **not exercised**. A genuine representative report is
   `01a1a011-b492-797c-964b-a626658a75aa`, applied 13 / signed lease 12. Its failed
   QoS detail is `read_at=... source=live candidate=12 verified=true ...`, while
   central's `generationHoldSafetyFailure` exempts only the exact dependent text
   `Rate policy could not be applied.`. Agent `attachServingRead` replaces that
   detail. [Raw checks/errors](follow-up-17-findings.json),
   [own-lease/report/incident joins](generation-holds-review.json), and
   [three original seam attempts](populated-holds.md) retain the evidence.
   The four post-restart samples retain `safety_status=expired`, so their dependent
   QoS text alone cannot establish pure classification either. That safety
   precedence is retained. The devstack 27 seam forwards observations unchanged and cannot manufacture
   the pure row. Needed: an owner-reviewed diagnostic/classification contract or
   a genuine supported path that produces it without changing signed documents,
   editing reports, or weakening independent-failure precedence. Relevant
   outside-owned files are `node/agent/serving_evidence.go` and
   `internal/nodes/generation_hold.go`; neither was edited. Stop only this item.

D-140 is qualified and its unchanged evaluator passes under prospective D-142.
The shared D-104 mechanism passes on netlab 26's fixture; no new fixture request.
Central's 30-second fallback passes from its original Step-1/no-alarm audit.
Historical isolation 15, isolation 16 and F14-01 remain failed as judged.

The nightly short-expiry block is **not established as expected on GitHub's
runner**. Status-only historical logs and zero artifacts cannot establish its
precise cause. The owned fix records actual reasons and uploads fresh records
on the next naturally scheduled run; no workflow was triggered. This is a known
diagnostic limit, not an allowlist exception or a request to infer a row.

All owned Docker inventories are zero. No install exceeded the 100 ms bound in
this run. No outside-owned edit or handover pin change was needed.
