# Follow-up 20 — r27 predicates and shared node writes

Dated 2026-11-03, **after** members 18 run 2 `50117c68`. This is an offline harness correction and replay proof. That run and every earlier run retain their original failed verdicts. No committed run record was edited, and no assembled run or Docker object was started.

r27 assigned M-2 (AT09 restart scope and lease classification) and M-3 (AT02 recovery measurement) to members. H-1 (false foreign-filter report on missing interface), M-1 (renewal delay after closed recovery), and L-1 (repeated quarantine/baseline writes) remain agent-core findings. No product, shared helper or network fixture was changed.

## AT09: restart scope and held leases

The timeline begins at `resumedAt`, with no two-minute lease lookback. Each non-serving report joins its own lease ID, generation and validity at the report's timestamp. Receipt order or the most recently issued central lease cannot substitute for the lease the node actually held.

A grant-less lease uses the existing D-067 revision 11 / D-126 composition predicate. A populated lease instead requires a coherent safety failure, with neither `awaiting_member_grants` nor a lease-stage error. Recovery accepts and records either `grant-less then populated` or `populated from first renewal`; unchanged-generation, serving, populated-lease, counted-byte, 90-second and query-completeness conjuncts remain.

Index 25 retains its name. When no grant-less lease occurred, it records **`blocked`**, the suite's not-exercised status, and never passes. The grant-less publication scenario in `populated_hold_test.go` still exercises D-067. Index 26 retains its name and records the observed sequence independently of whether D-067 was exercised.

A separate named subcheck, `restart without installed foreign filter reports no TC_FOREIGN_FILTER`, rejects any such report after the namespace recreation. Earlier tc objects cannot survive that restart. The run-2 replay accepts the populated-first lease sequence, records index 25 blocked, and **fails this separate H-1 check**. This is a proof of the new predicate, not a replacement verdict.

## AT02: application of confirmed D-135 and D-136

The orchestrator applies the confirmed decisions: “baseline → next successful Apply, every loss counted, recovery required, never waived” and “the bracket is reported, never waived.” **100 ms is an install budget, not an outage ceiling.** No new owner decision is requested.

Both `apply_rollback` and `apply_quarantine` link to the preceding failed Apply in the same producer run and boot. Each bracket opens at the first baseline/quarantine and closes at the first verified successful Apply with a populated lease and a written serving heartbeat. Repeated baseline calls remain in `baseline_calls`; overlapping intervals merge for packet accounting so no loss is counted twice. A verified zero-grant deny-all Apply is retained as a step inside the bracket, with separate install and subsequent deny-all intervals.

Every sender attempt remains in the denominator. Send bounds place interface-down losses inside the recovery interval even when node ingress sees nothing. A successful `foreign_tc_quarantine` call proves the link-down read-back; an `apply_quarantine` baseline callback alone does not. The down interval uses command bounds and retains the down-verification completion time, without inventing an exact kernel transition. Two of run 2's 983 missing-ingress sends fall during the initial lowering command; the other 981 follow its completion. All 983 are reported as not observed at node ingress in that verified interface-down quarantine interval.

Incomplete capture, missing send bounds, sequence gaps or ambiguous packet placement cannot produce a valid measurement. All losses must fall inside brackets; completed guards do not waive an outside loss. Forwarding must be observed after each close, before the next bracket or capture end. There is no outage ceiling and no zero-total-loss criterion.

The original check names remain. Index 190 asserts valid bracket accounting; index 191 independently asserts valid sibling recovery after every close. Both must pass. The revoked-source drop checks are unchanged.

### Run-2 offline replay, not a new verdict

| First baseline → serving Apply | Duration | Admitted-sibling losses | Forwarding after close |
|---|---:|---:|---|
| 404 → 408 | 218.707703 ms | 8 | observed |
| 416 → 420 | 202.757744 ms | 8 | observed |
| 553 → 593 | 25.156758057 s | 1,226 | observed |

The complete sender ledger contains 6,250 attempts, 5,267 ingress packets and 5,008 egress packets. All **1,242 losses** are counted: 983 missing ingress and 259 ingress without egress. All three failed Applies link to their `apply_quarantine` baselines. The offline result is valid accounting and observed recovery for 190 and 191, with zero losses outside the brackets.

Apply 584 remains inside 553 → 593. Its install lasts 198.343190 ms with 10 losses placed by send time; the following deny-all interval lasts 4.942222706 s with 239 losses. The entire step through the serving Apply lasts 5.140565896 s and contains 249 losses. These counts use send time, including sends not seen at ingress; they do not replace r27's ingress-time decomposition. M-1's delay remains visible and assigned to agent-core.

## Shared helper adoption and fallback

All five node writes use `labstack.Stack.WriteNode` through `nodeMutation`:

| Call site | Exact read-back | Receipt location |
|---|---|---|
| `serving_test.go`: publication | `NodePublished` for published intent and exact groups; `NodeFieldsMatch` for disabled intent and groups | publication command check |
| `serving_test.go`: name/region/capacity | `NodeFieldsMatch` over the entire requested patch | finite capacity/classification check |
| `populated_hold_test.go`: capacity stimulus | `NodeFieldsMatch` over the requested capacity | command check and hold record |
| `populated_hold_test.go`: clean capacity | `NodeFieldsMatch` over the requested capacity | hold record |
| `sibling_fault_test.go`: clean capacity | `NodeFieldsMatch` over the requested capacity | failed-Apply stimulus record |

The helper keeps one idempotency key and frozen body through bounded exact-conflict retries. Receipts retain actual HTTP status, ETags, digest, attempts and re-reads. “Already intended” leaves the 412 intact; publication locates a real operation for the current desired generation and still requires completion. No revoke write exists in this lane, so no `NodeRevoked` call is needed.

The AST audit rejects direct node writes, unresolved write destinations and new unaudited HTTP transports. Its exceptions are dispatch boundaries, not node writes: `live.api → request`, `live.request → NewRequestWithContext/Do`, device `fenceCall → api` (whose callers are audited), and the fixed-origin NetBird `nb → request`. Probe `Do` transports accompany audited GET-only constructors. Known member, policy, group, enrollment, webhook, agent usage and origin/payment-fixture paths are outside the node-write contract. There is no stale-node-ETag exception in this lane; existing expected-error tests are unchanged.

Devstack 33 (#441) answered both Follow-up 19 shared-helper requests. The grant fallback **remains enabled**, relying on the dated adoption note's explicit permission: same-plan eligibility is safe, though the extra POST creates another grant/audit and may extend validity to the fallback's one-hour end. Device/worker/fence checks and all restoration check names remain. A synthetic shared-restored state exercises that choice; later failures and normal expiry still restore normally.

## Proof changes and validation scope

The four r27 proofs were imported unchanged in standalone commit `e0844864`: the two TODO proofs failed and the two Held proofs passed before the fixes. M-2's copied old loop now calls the same `assessRestart` predicate as the live case, supplying explicit held lease IDs and timestamps; its grant-less-then-populated scenario and expected acceptance are unchanged. M-3 already called `siblingRecovery`; its fixture now explicitly sets the verified Applies and written serving heartbeat described by its original scenario. Its expected recovery and loss accounting are unchanged. Both Held proofs are unchanged.

New `TestF20_` tests replay the immutable records, reject capture gaps, unplaceable sends, outside-bracket loss, missing recovery/verification/lease/heartbeat and lease misclassification, exercise the actual exported node helper against a local HTTP stub, serialize receipts, and challenge the AST audit with bypasses. These tests require no live stack. Existing synthetic recovery coverage now supplies the verified populated Apply and heartbeat and expects bracketed loss to pass accounting. Historical capture facts and verdict records are retained.

### Requests to orchestrator

2026-11-03 — No new out-of-scope implementation request. Keep H-1, M-1 and L-1 with agent-core and schedule the next assembled run after the required fixes merge. Existing carried requests remain in [requests.md](history/follow-up-18-run2/requests.md).
