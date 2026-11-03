# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 37](../handover/follow-up-37.md) (2026-11-03 EST):** #424–#426: isolation readiness fixed locally; first scheduled exercise pending. Netlab 33 corrects L-1 and B1 citations additively. C1 remains documents only; no hitless hand-over. r24 H-1/H-3, F30-01 and r20 H-1 stay open on main. Ten owner requests.
**Previous cut — [follow-up 36](../handover/follow-up-36.md):** #422–#423 at d7867928; 16 subtests, two final rows, ten frozen inputs, exact r24, 174 netlab files (12 gzip originals), two PR excerpts and separate source audit; 96 findings, 47 decisions, ten requests.

## Follow-up 37 acceptance snapshot (2026-11-03 EST)

Cut **#424–#426**, main **792e3e41**; no unbriefed PR inside the range.
[Evidence and ten owner requests](../handover/follow-up-37.md).

| Lane / PR | Result and boundary |
|---|---|
| Isolation 18 (#424) | Scheduled 41640196805 failed correctly: two listed and two unlisted blocks. One harness defect, restart without readiness; fixed. Local 9f1bc0ff: AT10 9/0/0, AT11 2/1/0, AT12 10/0/0, AT13 6/1/0; short whole-node expiry qualifies under D-142. Only operator rows block. Previous night's 41456092596 AT10 pass used inherited readiness and is not qualified. First scheduled exercise of the fix is pending. |
| Handover 36 (#425) | #422–#423 at d7867928; 16 subtests, two rows, ten frozen inputs, exact r24, 174 netlab files / 12 gzip originals, two PR excerpts, separate audit; 96 findings / 47 decisions / ten requests. Lock citation corrected in three documents and one pin; approved refresh changed 53 digest values only. |
| Netlab 33 (#426) | One additive corrections.md; netlab 32 trees unchanged. L-1 confirmed in part: lock claim rejected, empty eligibility retained. M-3's B1 transport characterization unsourced; no transport tuning in C1 remains required. 16 citation rows: 14 verified, two corrected. Four continuations recorded in the cut. |

[97-row findings register](../handover/README.md#findings-register-at-the-follow-up-37-cutoff).
Isolation F18-01 is fixed in the harness; its runner exercise is pending.
AT11's native platform matrix, AT13's full fault matrix and Isolation F17-01
remain carried. The raw r24 archive and 94 unaffected findings stay unchanged.
Agent-core 37–41 were delivered and held; H-1/H-3 fixed on the unmerged branch,
r25 running at 8cc27609 (status only). Netlab F32-01 goes to agent-core 42.
r24 H-1 stays open known behaviour on main; F30-01 stays known behaviour and
r20 H-1's restart gap stays open. Follow-up 36's reload guidance is unchanged:
no hitless hand-over is implemented, no date or C1 benefit promised. Production
keeps the current guard pending the owner's G5/G6 decision. All 47 decisions
remain unchanged through D-144; every historical failure stays failed.
