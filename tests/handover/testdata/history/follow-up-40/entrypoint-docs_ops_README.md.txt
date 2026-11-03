# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 39](../handover/follow-up-39.md) (2026-11-03 EST):** #428–#429 at 1b87ab32: agent-core 37–43 merged; r26 archived with zero findings. r24 H-1 and r20 H-1 fixed; F30-01 stays known behaviour. Production keeps the guard under D-145; C1 unimplemented. D-146/D-147 await netlab 34; F11-01 is upstream-dependent. All ten owner requests answered; 53 decisions.
**Previous cut — [follow-up 38](../handover/follow-up-38.md):** #427 at 77626af4; 12 subtests, one final row, ten frozen inputs, one PR excerpt, exact r25 and separate audit; 99 findings, 47 decisions, ten then-open owner requests.

## Follow-up 39 acceptance snapshot (2026-11-03 EST)

Cut **#428–#429**, main **1b87ab32**, immediately after #427; no unbriefed PR.
[Evidence and ten answered owner requests](../handover/follow-up-39.md).

| Item | Result and boundary |
|---|---|
| Handover 38 (#428) | #427 / 77626af4; 12 subtests, one row/PR excerpt, ten frozen inputs, exact r25 and separate audit; 99 findings / 47 decisions / ten requests. Two commits, 28 files; 258 digest values only; no new request. |
| Agent-core 37–43 (#429) | r20–r25 judged with corrected attributions and rejected sub-claims retained; Netlab F32-01 fixed. Healthy unchanged enforcement is retained after fresh verification; renewals/model changes and required repairs still install. Refused-restart accounting fixed with provenance/settlement gates. F30-01 unchanged. |
| r26 archive | 19,545 bytes, unchanged; 0 Critical / 0 High / 0 Medium / 0 Low. Reviewer recommends merging 37–42. Orchestrator merged 37–43 under provisional D-150; both r25/r26 proof files now on main. |
| Timing boundary | 43's two windows qualify no leg; pooled A and B each n=0 for all five spans. No comparison/attribution. Separate ordinary B lab passes 13 scenarios; D-150 routes post-merge measurement to agent-core 44. No 44 result in this cut. |
| Owner decisions | D-145 keeps current guard/C1 unimplemented; D-146 TCP MSS clamp and D-147 narrow RELATED fragmentation-needed are being implemented by netlab 34, not on main here, no date. D-148 leaves F11-01 open upstream-dependent, no product change; D-149 dismisses development dependency risk. Eight confirmations; all ten requests answered. |

[99-row findings register](../handover/README.md#findings-register-at-the-follow-up-39-cutoff):
30 rows updated, 69 byte-identical. **53 decisions**; original confirmation
substance and ten request texts retained. No hitless hand-over or C1 benefit.
Netlab 34 and agent-core 44 were running at briefing, status only and outside
this cut. #424's first scheduled exercise remains unrecorded by these PRs;
AT11 / AT13 operator rows and Isolation F17-01 remain carried. Every historical
failure stays failed. Earlier snapshot sections below describe their own cuts.
