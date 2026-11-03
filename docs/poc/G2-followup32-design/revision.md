# C1 revision after r24 — owner decision still pending

2026-11-03. Design only; **production retains the unconditional deadline guard**.
This addendum supersedes the specific passages below in the immutable
[Follow-up 31 design](../G2-followup31-design/hitless-guard-design.md). It implements
none of C1. Main's independent enforcement defects and proofs are documented in
[the enforcement judgment](../G2-followup32-evidence/judgment.md).

Inputs: [roadmap row](../../ROADMAP.md),
[PR #421's orchestrator reading](https://github.com/TED-H-com/Freedom-VPN/pull/421),
r24 `ba5d3004` (reviewed base `52a7e59b`), and
[D-133/D-134/D-135/D-141](../../handover/README.md). D-141's rejected intersection
serving representation is distinct from C1's temporary deadline intersection;
neither has implementation approval from this task. Handover owns r24's archive.

Read [revised protocol and prerequisites](protocol.md), then
[frequency, exposure and the owner's question](owner-decision.md). The original
[proof plan](../G2-followup31-design/proof-obligations.md) and
[comparison declaration](../G2-followup31-design/d133-comparison.md) still apply
except for these explicit corrections. A future comparison must re-freeze A on
the reviewed production fixes, not reuse F31's old main hash. No comparison or
reservation is authorized in this session.

| Review item | Judgment | Revised text and reason |
|---|---|---|
| H-1 | Confirmed by code; agent-core work | No-op Step currently calls Apply. Fix before assessing C1's benefit; cadence is recalculated in owner-decision.md. |
| H-2 eligibility | Confirmed in part | Preserve eligible identities across rate-only fair-share changes with stable class and non-tightening ceilings. `min(fair, ceil)` means not every churn changes every rate; removing A2 while A1 remains does not change member count. Include independent-member churn in AT13 proofs. |
| H-2 tc | Confirmed main defect | Production fix separates preparation and retirement. C1 inherits that order; no class deletion while old nft can classify into it. |
| H-3 | Confirmed outcome, corrected explanation | `memberTimeout` DOES cap by binding.ValidUntil. The earlier member-wide deadline collapse and first-binding service/node ends still truncate siblings. Require per-device deadlines and a separate verified-serving record from agent-core. |
| H-4 | Confirmed main defect | State-independent baseline and cancellation-independent cleanup are prerequisites; no C1 bank-aware implementation here. |
| M-1 | Confirmed design correction | Delete old bank in release; no retired-bank exception after success. Delayed fallback installs are fenced by lock/child reaping, not an old-bank reference they do not have. |
| M-2 | Confirmed in part | Post-shape clock paths now attempt rollback. Alleged currently reachable fatal verified result is rejected with source evidence and input regression. |
| M-3 | Confirmed by tooling inspection | Require real classifier/decorator tests and ctunku/ctunkd deltas; prohibit B1 transport changes in acceptance. The copied enforce test is not tooling proof and is dropped. F31 already mentioned an adapter; it was not implemented. |
| L-1 | Confirmed | Correct lock location and specify the empty eligible-set transaction shape. |

The original [G1–G10 ledger](../G2-followup31-design/current-guarantees.md) remains
the numbering authority: **G5 crash and G6 late commit** are the owner's trade,
with G4/G8 containment consequences. r24 uses different G labels in its “held”
section; it must not accidentally rename the decision. §8.3 concerns IPv6/NAT;
QoS/host reserve obligations come from §11. No acceptance predicate is weakened.

### Requests to orchestrator

1. **2026-11-03 — owner / decision register in the orchestrator's private
   repository:** relay the final paragraph in owner-decision.md and record the
   G5/G6 answer. Until then keep the unconditional guard; no C1 rollout.
2. **2026-11-03 — agent-core, `node/agent/engine.go`, `decide.go`, `plane.go`,
   `restart.go`, `serving.go`, and their tests:** fix H-1 and the H-3 device
   deadline collapse in the assigned later follow-up. C1 additionally needs
   the verified-serving provenance contract in protocol.md. Import r24's agent
   and watchdog tests in their owning lane, not this one. Review the nanosecond
   jitter calculation when using cadence figures; no change is made here.
3. **2026-11-03 — agent-core and updater/installer owners:** after owner approval,
   inventory all bank readers/baseline writers and old-binary rollback paths,
   including `node/watchdog/*`, `node/updater/*`, `packaging/*` and
   `tests/netlab/agent/*`. Unknown/interrupted format must close before adoption.
4. **2026-11-03 — acceptance owners / orchestrator:** native arm64/kernel proofs,
   cross-lane consumers and a quiet D-133 comparison remain separate assignments.
   Netlab's future C1 tooling prerequisites are specified, not changed here.

Follow-ups **25–31 remain byte-identical**. Historical B1 stays FAIL and
unexplained; Follow-up 30's live populations stay host-unqualified. This design
creates no new availability guarantee, operator procedure or accepted C1 result.
