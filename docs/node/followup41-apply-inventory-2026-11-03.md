# Follow-up 41: Apply-effect inventory — 2026-11-03

Inventory of `b17c8c17`, committed before the H-1 fix. Scope: §8.1–§8.3,
§10.2–§10.9, §11, §13.9, §14.7, §16.3; ISO01/ISO02, MTR01–MTR04,
AT10/AT13 mechanisms. This does not implement C1 or change its guard.

| Current every-tick effect | Retention requirement |
| --- | --- |
| Decide, probe/DNS refresh, lease/cert/bootstrap synchronization | Still run every Step; signed times, ValidFrom/Until, rate-policy windows and current local configuration rebuild the candidate |
| Node lock, updater latch and checkpoint invalidation | Retention is inside the existing lock after rereading the latch. No new admission means no checkpoint invalidation; pending hand-off/checkpoint state prevents retention |
| Load live nft state/counters and tc ownership; detect resets | Existing serving verifier checks live nft/tc, generation and metering marker on every retained tick. Continuous meter samples detect counter resets independently of Apply |
| Render and replace HTB roots/classes/leaves/rates | Compare datapath and the full QoS including actual assigned classes. Unchanged verified tc objects have no refresh TTL |
| Render allowed/deny/host/resolver sets, chains, SNAT, counters and guard; release guard | Compare the entire renderer model, with only elapsed clock samples normalized to the same anchor. Unchanged verified sets/chains need no rewrite. Guard is an installation fence, not periodic maintenance |
| Member element lifetime refresh | Last installation alone carries the absolute authorized deadline; see arithmetic below. Lease renewal and any changed device/member deadline must Apply |
| Probe sets and probeSeconds hold | Compare full Probe including source, destinations, resolver flag, expiry and cap. Existing live sets may be held when remaining lifetime is unknown and a new max window would extend authorization. Skipping leaves those bounded elements intact, never renews them. Challenge install/clear remains its own path; pending cleanup prevents skip |
| Revocation planning, durable conntrack reservation, deletes, retry accounting | Any pending address, attempt reservation, block, or computed revoke takes the normal enforce path. Successful past Revoke is a consumed action, not persistent kernel model state. Never skip a retry |
| Quarantine release, member link restoration and member-path capture/repair | Any pending quarantine, missing path or repair forbids skip. A retained tick still checks the saved member topology under lock. Failed ownership/path reads use existing quarantine/withdraw rules |
| Pre-Apply usage identity hand-off, closed-epoch preparation and durable intent | Retained bindings must match active meter identities; no unsettled retained interval, closed/final epoch, persistence failure or changed intent/epoch is eligible. No new admission means no new intent is needed when identity and epoch already match |
| Save retained model, ACK snapshot, QoS assignments, counters and addresses | Existing durable snapshot is still the same. A failed previous write sets storage/apply/pending state and cannot skip. Generation and effective assignments participate in equality |
| Kernel reset capture/nudge, epoch switch, sampler resume | Sampler remains running, health is still read, and epoch reports still publish. Pending nudge or changed epoch prevents skip; withdrawn/closed/paused-hand-off state must resume through normal Apply |
| Apply measurements and post-Apply verification | No fabricated Apply/guard record on a skipped tick. Live verification and its measurement/readback record still occur; existing observation and health answer paths remain |
| Heartbeat before lock release, end-of-Step live read and watchdog expiry | Fresh retained verification precedes its heartbeat; end-of-Step verification remains. Keep the existing absolute PolicyExpires; never slide expiry with a heartbeat. Changed policy expiry forbids skip. Busy/failed lock cannot count as the required retained verification |

The comparator must include generation, IPv6 mode, every binding and revision,
all member leases, both clock anchors, datapath/SNAT, resolvers, deny/extra-deny
and host prefixes, QoS parents/host/member rates and classes, Probe, and the
separate PolicyExpires. MeteringVersion is a compiled renderer constant and
is checked by the live generation/metering marker verifier; changed binaries
cannot adopt a mismatching marker. A retained audit model alone is insufficient.

## Element lifetime arithmetic

For a device, let D be the minimum of its own ValidUntil converted using the
lease's monotonic-derived clock and the member limits (lease, service end,
node window/offline/policy cap). At render time R, Apply installs
`floor_ms(D - R - 100ms)`. With successful guarded commit C in `[R,R+100ms]`,
expiry is `C + floor_ms(D - R - 100ms)`, in `(D-101ms,D]` absent an extra
explicit timeout cap. No ten-second refresh is involved. H-3 must be fixed
first so a short sibling cannot replace D with its deadline. Agent-generated
member leases have no extra relative TimeoutCeilSeconds cap.

Thus skipping cannot cause the periodic-refresh dependency or multi-minute
premature sibling expiry in H-3. Literal expiry never before the signed time
is NOT a current D-104 guarantee: the 100ms safety reserve and millisecond
truncation deliberately close slightly early. This existing bound is kept.
Deny/host/resolver sets have no timeouts. Probe elements have their own expiry,
whole-second truncation, explicit cap and hold rule; they are intentionally
short-lived and never depend on ordinary member Apply to stay alive. No new
lifetime measurement is claimed.

## Boundary handling and failure

A ValidFrom crossing changes the candidate bindings; ValidUntil/member expiry
removes them, including a single expired sibling. The current policy-expiry
withdrawal must be followed by redecision on that same tick, so a surviving
sibling is not deferred an extra poll. Read failure/drift may repair or withdraw
according to the existing rules; no cached serving claim substitutes for a
live read. Model equality is conservative: an uncertain difference Applies.

## Pending state added in 37–40

Retained accounting provenance, `retainedPending`, `usagePersist`,
`epochClosed` separate from `finalDone`, barrier identity, active meter bindings,
restart intent/epoch, baseline certainty, failed candidate/apply error, storage
health and kernel epoch nudge all remain gates. Boot's adoption/settlement
worker and close ownership are unchanged; Step does not race Boot or Close.
A reset-only contiguous intent remains valid restart evidence (40), but an
ordinary retained tick must not silently postpone a changed active epoch.

## Implementation clarification — 2026-11-03

The locked retain branch is inside `enforce`, after existing conntrack gates
and the updater latch, before pre-Apply work. `latch` still removes an orphan
meter checkpoint on every pass; the inventory's checkpoint row refers only
to skipping the redundant pre-admission invalidation. Active-epoch equality
adds a read-only usage Head lookup; it does not sample or write a new intent
on an unchanged tick. Thus 40's historical statement of no extra healthy-pass
SQL read does not describe 41. Pending epoch/intent work uses normal Apply.

`SystemPlane.VerifyServing` now wraps its executor with
`servingPresenceExecutor`: the existing nft JSON snapshot is retained for an
allow-element completeness check when the existing precise lifetime read is
reached. It adds no command or write, preserves prior nft/TC failure precedence,
and records completeness failure as a failed deadline verification. Elements
within the existing 101ms early-expiry allowance may legitimately be absent.
`enforce.VerifyServing` itself is unchanged. Failed live reads still withdraw
or repair using existing rules; foreign tc remains owned by its installer.

The first expired-device/member boundary still takes the existing conservative
node withdrawal and usage settlement, then redecides on that same Step. It does
not defer surviving devices to the next poll. C1 or hitless boundary transitions
are not implemented. End-of-Step verification, watchdog expiry and measurement
records remain independent of whether Apply ran. No Apply record is fabricated.
