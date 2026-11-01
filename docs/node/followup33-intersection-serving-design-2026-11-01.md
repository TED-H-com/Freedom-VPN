# Intersection serving: implementable proposal for r19 — 2026-11-01

**Design only, pending the recorded D-139 representation decision. Option (c)
remains production behavior.** The companion
[executable reference](../../node/agent/followup33_intersection_spec_test.go)
is test-only. This proposal grants no implementation approval, changes no
deadline, reserve, ceiling or acceptance threshold, and claims no new AT pass.

The intended benefit is continuity for a previously admitted, unchanged sibling
while another device is revoked. It does not guarantee continuity across every
restrictive policy change. In particular, this first implementation profile
falls back to the baseline when tighter QoS needs a tc mutation, DNS filtering
content changes, or topology/accounting identity changes. Performing those
operations before revocation would lose the property being sought.

Read with [follow-up 31](followup31-generation-change-availability-2026-11-01.md#alternatives-and-invariants-for-review),
[follow-up 32](followup32-shutdown-hold-observations-2026-11-01.md), and
[r18 “What held”](../reviews/r18-generation-change-review.md#what-held).
The governing requirements are spec §5.1–§5.2, §8.2, §10.2–§10.3,
§10.7, §11.3–§11.5, §14.7 and §16.3: ISO01/02/04, ENT01/02,
MEM03, MTR01/04 and FUP02/04. D-135 rollback, corrected D-136,
D-106 evidence and D-134's 100 ms install bound / 1 ms granularity remain.

## 1. Names, invariants and implementation boundary

* **O**: last fully verified serving desired document, generation `g`, its
  original signed lease **L**, and the actually installed/verified model **A**.
  An ACK of a baseline is not this premise. All three must exist and agree.
* **N**: authenticated, non-replayed new desired document, generation `h>g`.
  A signature failure never creates an intersection. Follow existing document
  rejection/expiry behavior; do not manufacture a verified N.
* **S**: the locally derived subset of O ∩ L ∩ A ∩ N. It has a content hash and
  a transition ID, **not** a new signed generation. Neither S nor a successful
  subset read authorizes all of h.
* **E**: the first revocation transaction after accepting N, at the place
  where option (c) would withdraw. No lease request, tc change, DNS call, probe,
  observation POST, new lock retry window, or deferred durable write may be
  inserted ahead of E. If a ready bounded plan is unavailable there, baseline.
  Read-back is after E; a failed/unknown commit is not successful revocation.
* **D(b)**: the earliest original same-boot monotonic limit for binding b,
  further reduced by N. A later response, render, retry or report cannot move it
  forward. Known revocation must act at E, not wait for D(b).

Use one serialized node mutation owner, the existing shared node lock, and an
in-memory cancellation epoch for freeze/revocation. Compare identity, target
hash and that epoch immediately before every kernel commit and report/ACK.
Freeze notification latches the epoch before waiting for that owner. Once it
wins the order, no queued intersection or full Apply can release admission.
If it arrives after an irreversible kernel commit, the next transaction is
baseline, with no further open commit in between. This is a linearized ordering,
not a claim that two kernel transactions or SQLite and nft are atomic together.

Existing `Step` calls `syncLease`, certificate/bootstrap work, probes and meter
work between desired acceptance and `enforce`. The eventual implementation
must intercept an accepted restrictive N at that boundary and execute E before
those calls. Merely changing `leaseGenerationStale`, `Decide.Serve`, or
`recoveryDecision` is not this design. No new network wait belongs on that path.
Prepare against the already retained verified model during parsing; use the
existing acceptance durability boundary to record intent. On a slow/failed
acceptance write, abandon S and withdraw within the existing local budget.

## 2. State machine

The states below belong to one transition, not to publication intent. There is
only one live transition. A second desired generation never starts a chain of
intersections or recovers a previously excluded identity.

| State | Entry and allowed work | Exit / guard |
|---|---|---|
| F: full serving g | Current same-generation authorization, model, live nft/tc/path read, meter and watchdog qualify. | Receipt of verified h enters P. Any existing safety failure takes B/Q. |
| P: restrictive desired accepted, intent durable | Pin O/L/A and N, invalidate ordinary same-boot adoption in the same SQLite transaction as accepting N. No new member may enter. Old kernel rules exist only until the immediate E call. | If the §3 meet and §6 realization proof are ready, commit E → K; otherwise commit baseline → B. Never wait for renewal here. |
| K: subset kernel commit, not yet verified | Kernel can carry only S, including original timeouts; no serving assertion or h ACK. Finish the bounded nft/tc/path/meter read and exact cleanup. | Complete proof + durable active checkpoint → I; failed/late/partial read or write → B/Q. The verified conntrack exception below is the only tolerated cleanup error. |
| I: verified intersection | Retain S unchanged, live-read before/after control work and before heartbeat. Renew for h using the normal one-attempt-per-Step and Retry-After rules. Request still reports ACK g; issuer already signs its desired generation. | Fresh valid populated h lease → A. Replay/transport/budget failure remains I only while all original limits hold. Empty h lease → B immediately; a valid restrictive grant update can never be ignored in favor of old grants. |
| A: full h candidate installing | Pin new lease separately from L; check current target/hash/freeze epoch, mark `full_pending` durably, close S with verified baseline/handoff, then run the existing guarded full Apply. All additions remain denied until this authorized full install. | Successful Apply + independent live read + path and meter verification + durable full checkpoint → F(h). Any failure → B/Q, never restore S or O. |
| B: baseline | Zero member admission; keep control, account final bytes, keep revocation backlog. May use the existing non-serving h ACK semantics after baseline verification. | Reacquire current matching lease and complete a full verified Apply. No return to I. |
| Q: quarantine | Foreign tc, topology bypass or failed containment requires existing containment plus baseline; preserve first rejected evidence. | Existing repair/preflight/full authorization workflow; never restore S. |
| X: identity frozen/revoked | Clear S and candidate admission atomically with the baseline transaction, invalidate transient models, retain incarnation latch and uncertain-response journal. Suppress ACK. | Revoke/re-enroll as today; later desired/lease/false response does not unfreeze. |

Expiry rules apply at **P, K, I and A**, before and after every operation. In P,
do not include expired bindings. In K/I, the kernel independently expires each
admission; a heartbeat cannot refresh that timeout. At the earliest retained
binding deadline this initial profile withdraws the whole subset (B), rather
than creating an unreported new subset hash. Using the earliest member/device
limit for a shared member allowset is allowed and conservative. Node/policy,
clock, meter, release, publication, updater-barrier or isolation failure always
closes everyone. In A the new full candidate has its own legitimate limits,
but L/S cannot be extended while installation is pending. A read straddling a
deadline cannot publish a fresh success. The deadline reserve is not a grace.

Superseding verified desired `j>h` from P/K/I/A takes baseline immediately,
persists the new target, and requires a full j Apply. There is no S(g,h,j), no
subset renewal under another g lease, and no permission to re-add a device
removed at h. Lower-generation and same-generation/different-content responses
are rejected by anti-replay/equivocation checks. A replay of the exact current
document does not reset anything.

Lease handling in I is equally explicit: an exact replay of L keeps its anchor;
an unauthenticated, replay-rejected or wrong-identity response grants nothing;
an authenticated newer lease for the wrong generation cannot replace L or
promote S and takes baseline. Any newly accepted restrictive/empty member grant
set also closes S. Only a valid populated lease for the still-current h and
its authorization revision can enter A; its signature, identity, anti-replay,
time and member checks all remain mandatory. A fresh g lease is not permission
to prolong S. No retry/backoff path edits the pinned envelope or deadlines.

Graceful stop in P/K/I/A withdraws; it must not call ordinary
`preserveServingOnStop` successfully. Same-boot restart, reboot and crash recovery
of any transition state install baseline **before** interpreting retained
state, reacquire authorization and perform full Apply. Even a complete I
checkpoint is evidence only. To cover a crash after authenticated receipt but
before the intent write, the first profile **disables all same-boot adoption
while the feature is enabled**, including completed F checkpoints. The durable
feature setting is read before Boot's adoption branch. This deliberately costs
restart availability; full-generation adoption remains today's behavior only
with option (c)/the feature disabled. It is part of Q4's cost review, not an
unmentioned exception to the crash table.

## 3. Exact subset rule and representability

### 3.1 Identity and signed revision witnesses

Require O, L, A and N to refer to the enrolled node/incarnation, with L and A
at g and N at h. L's authorization generation must equal O's authorization
policy generation. N's authorization and binding-set revisions must not go
backward. An unchanged binding-set revision with changed binding contents is
ambiguous and takes baseline. The set revision may advance because another
member changed; it is not the revision of an unchanged binding.

For each candidate device require exact equality of member ID, device ID,
opaque peer ID, the **complete canonical set of IPv4 host addresses**, and its
per-binding revision across O/N/A. Canonical bare IPv4 and /32 mean the same
host; order is irrelevant. A removed address, an extra address on that device,
a changed peer, or an increased/decreased binding revision excludes that entire
device. A newly added device is excluded even if L already grants its member.
Device/peer/address duplicates, non-host sources, IPv6 admission in this IPv4
profile, missing identities or unparseable fields are global ambiguity: baseline.
Multiple distinct devices of one member share one class per direction.

L must actually grant that member. Its nonzero `entitlement_revision` must
equal the signed per-member witness in **both** O and N. A missing member
witness or a different revision excludes every device of that member. A
missing witness document, duplicate witness, unknown extension semantics or
inconsistent member-rate selection takes baseline. Never infer entitlement
from member presence, a binding-set revision, or a rate-policy generation.

**Blocking contract gap:** today's `DesiredState` has no per-member entitlement
revision. `MemberLease` has it; `MemberRatePolicy.Generation` is a different
quantity. This design therefore proposes `intersection_constraints` inside the
signed desired object (§5). The compiler must obtain its complete eligible
member set, revisions and service ends from the same authoritative snapshot as
the bindings. Absence means unsupported, hence baseline. This is restrictive
evidence, not a new grant. If owners reject it, this design cannot safely
implement the revision check and option (c) remains the answer for that item.
The first eligible O must itself have been fully applied with this witness;
upgrading a process does not retrofit witnesses into old signed documents.

### 3.2 Policy algebra

After authenticating and normalizing both documents:

```text
admission = exact old/admitted identities also present unchanged in new
denied = old.denied ∪ new.denied ∪ immutable baseline denials
destinations = old.allowed ∩ new.allowed          # semantic IP sets, not strings
resolvers = old.designated_resolvers ∩ new.designated_resolvers
node_ceil[direction] = min(old_limit, new_limit, actual_old_ceil)
member_ceil[member,direction] = min(old_limit, new_limit,
                                  actual_old_ceil, node_ceil[direction])
```

Denial wins over ordinary destination admission. The existing designated-DNS
exception remains exactly TCP/UDP 53 to retained resolvers and cannot open
their management service or subnet. CIDR overlap uses address-set intersection
(a /24 ∩ its /25 is the /25); union must not drop either denial. There is no
raw domain/destination list in observations or logs. Changed DNS *filtering
content*, unresolved resolver-document reference, policy interpreter/version,
IPv6 mode, host/overlay ownership, NAT/route/namespace/interface topology,
meter epoch/period, host reserve or class identity takes baseline. Different
references are acceptable only when their fully authenticated semantic content
is identical; unresolved reference equality is not a content proof.

Rate-policy selection must be unambiguous for both directions over the whole
remaining interval, including priority/mode/period. Normalize the effective
policy using its signed interval, member cap and node cap; null/shadow must be
resolved to the finite node-limited ceiling, not zero or infinity. Take minima
independently for upload and download. Stop at the next policy interval boundary
if the selection could change. Zero/missing/invalid ceilings take baseline.
Retain HTB class assignments, rate guarantees, fq_codel, host reserve and both
node parent ceilings; removing a member must not recalculate a larger sibling
rate or ceil. Actual installed ceilings must already obey O. A corrupt old
shape invalidates the premise, even if N happens to permit it.

**Realization at E:** this first profile performs only nft restriction and
does not reshape tc. If any surviving member or node needs a lower actual
ceiling, use baseline at E and full Apply later. The reference returns the
mathematical minimum for inspection but marks that candidate `Baseline=true`;
it authorizes zero traffic. A previously lower installed ceil can be retained
when it already satisfies both policies. There is no temporary overshoot,
nft rate-policing replacement for HTB, or tc-first optimization. Empty resolver,
destination or binding intersection also selects baseline. A policy whose
semantics cannot be represented by the restriction transaction selects baseline.

### 3.3 Deadlines

Let `t0=L.issued_at`, `m0=L.original_request_mono_ns`, with the original boot.
All signed timestamps are converted by `m0 + (timestamp − t0)`, using checked
integer duration arithmetic. Never use N's receipt time as a fresh anchor.
Reject overflow, missing times, negative validity windows and untrusted clock.
Validity starts may precede t0; the interval must contain the current anchored
time. Check the later old/new start and use half-open validity.

```text
D(b) = min(
  A.original_binding_deadline(b),
  m0 + (L.node_expiry - t0), m0 + L.max_offline_seconds,
  m0 + (O.expiry - t0), m0 + (N.expiry - t0),
  m0 + (L.member_lease_end - t0), m0 + (L.member_service_end - t0),
  m0 + (O.member_service_end - t0), m0 + (N.member_service_end - t0),
  m0 + (O.binding_end - t0), m0 + (N.binding_end - t0),
  m0 + (O.rate_selection_end - t0), m0 + (N.rate_selection_end - t0))
```

Neither N's longer validity nor a later old-generation lease extends D. For
multiple devices sharing a member timeout use the minimum of their D values.
At render, subtract the **existing 100 ms reserve once**, round down to the
existing 1 ms granularity, and omit nonpositive allowances. The verifier checks
the absolute signed D without a second reserve subtraction. Preserve existing
earlier kernel expiration where possible; recreating an element must prove
its resulting expiration is no later than the retained old expiration and D.
Expiry readbacks are conservative bounds from the read bracket, not exact
kernel commit times; a late read-end timestamp cannot become a fresh timeout
anchor. If that bound is unavailable, the candidate is ineligible.
The reference models signed D and reserve eligibility; real kernel timeout
read/conversion and install-overrun behavior belong to netlab's proof.

## 4. Provenance and crash consistency (schema plan only)

Plan a local agentstore schema v4 in `node/internal/agentstore/store.go` and
typed operations in `state.go`; no migration is written in this follow-up.
Keep the existing desired anti-rollback record. Add one `serving_transition`
row, with a version, transition UUID, phase (`intent`, `active`, `full_pending`,
`closed`), node/incarnation/kernel boot/process run IDs, g/h, old desired hash,
old immutable signed envelope, old lease ID/generation/envelope/hash,
**original** request monotonic anchor and signed issue time, old applied model
hash/checkpoint, new desired hash/envelope, subset hash/normalized model,
per-binding absolute deadlines, original kernel expirations, class assignments,
counter identity map, exact pending revocations/retry count, and actual
verification timestamps/results. Integers are checked decimal TEXT in new
columns, not SQLite signed generation counters or JSON floating point.

Old/new desired and lease hashes are SHA-256 of RFC 8785 canonical `signed`
objects; retain envelope bytes to reverify signatures. Subset hash is SHA-256
of a domain/versioned JCS object `kind=fvpn-intersection-v1`: transition identity,
node/incarnation/boot, g/h, old/new desired and lease hashes/ID/anchor, normalized
identity tuples/revisions, effective policy sets, both-direction ceilings and
retained rates/class assignments, absolute deadlines and kernel-expiry caps,
topology/meter content hashes, reserve/granularity. Sort set-valued arrays by
canonical identity/address; decimal strings encode large integers. Exclude
read timestamps, current remaining timeout, packet counters and retry counts
from this immutable model hash. A later verifier reads decreasing timeouts
against the fixed caps; it does not mint a new hash each tick.

This local normalized policy material has the same private authorization scope
as retained desired state, not observed traffic. Do not export destination
lists, query names, packet contents, raw kernel listings or addresses in error
text. Public observations carry hashes, IDs, times, counts and sanitized checks.

At desired acceptance write N and the pinned intent plus `adoption_forbidden`
atomically in SQLite WAL/FULL. This uses the existing acceptance write, not an
extra persistence round trip before E. Active checkpoint, revocation ledger
and immutable observation outbox entry are one later transaction. Failed disk
work closes admission; it is never treated as permission to retain an
unrecorded S. The global time anchor may advance on legitimate renewal, but
the transition's L anchor never changes. Store a new h lease as a separate full
candidate while serving S; report L until the full commit.

Never mutate/re-sign L, rewrite its generation, or reuse the temporary
generation rewrite in `recoveryDecision`. That existing function constructs
deny-all recovery; it is not a cross-generation authorization primitive.

| Boundary / crash window | Possible kernel and durable state | Safe recovery |
|---|---|---|
| Before N acceptance commits | O or partially received N; ordinary F checkpoint | Feature-enabled Boot always baselines, reacquires and full-Applies. No durable knowledge of N is assumed; kernel D/watchdog still bound O before recovery runs. |
| N+intent transaction commits, before E | O still installed; N and adoption veto durable | Boot baselines first, retains O identities for final accounting/cleanup, reacquires h or newer. Never adopt O. |
| E command starts, result unknown (including timeout/process death) | O, S, guarded S or baseline; intent durable | Baseline/containment; inspect counters without admitting. Unknown outcome is failure, never infer atomic success from command start. |
| E committed, before read-back or active write | S or guarded S; intent durable | Baseline; reconstruct pending deletes from O minus S. No restart adoption. |
| Read-back succeeds, before active transaction commits | S; intent only, possibly some conntrack progress | Baseline; repeat exact idempotent deletes. Preserve old counter identities. No ACK h. |
| Active write fails/torn, or succeeds before heartbeat rename | S; intent/active/unknown row, old heartbeat | Baseline. SQLite error or torn heartbeat cannot authorize. |
| Active + heartbeat durable, before/after observation POST | S; immutable outbox may be pending/accepted/in-flight | Baseline on restart. In-flight response uncertainty retains freeze gate; replay exact report ID/body, never freshen its time. Central receipt is not a new serving grant. |
| Conntrack deletes partly succeed, before retry-ledger write | S; old backlog/count | Baseline after crash, replay exact deletes; persist attempt reservation before each batch so crashes cannot erase the retry bound. No address reuse while pending. |
| New h lease durable, before `full_pending` or full kernel commit | S and L checkpoint plus separate h lease | Baseline; reacquire then full Apply, never use h to restart S. |
| Full guard/candidate commit or guard release, before full verification | Guarded h or h; `full_pending` veto | Baseline; h installation is not yet a full ACK. Account all admitted bytes. |
| Full verification/path succeeds, before full checkpoint transaction | h kernel; `full_pending` | Baseline, not h adoption. |
| Full checkpoint commits, before new heartbeat rename | h; checkpoint+ACK h, old intersection heartbeat | Baseline; mode/hash/transition mismatch prevents adoption. |
| Full checkpoint and full heartbeat durable | Verified h; transition retired atomically with checkpoint | Feature-enabled restart still baselines/reacquires; completed h does not exempt this profile from the no-adoption rule. |
| Freeze arrives at any boundary | Memory epoch latched; baseline and freeze intent/SQLite writes may be incomplete | Existing freeze sidecar and observation in-flight journal veto admission; transition veto adds another barrier. Baseline first; uncertain persistence never clears freeze. |
| Baseline/quarantine commit or closed-record write is incomplete | Unknown/closed kernel, transition still vetoed | Retry containment/baseline; retain cleanup/accounting evidence, no intersection recovery. |

All local mutations share follow-up 32's bounded Step/cleanup/Close allowance.
No new five-second cleanup budget is started for each transition stage. A failed
baseline cannot be labeled successful containment: keep quarantine pending,
emit failure and rely on the independent watchdog/kernel deadline backstops.
The crash rows do not promise instantaneous removal after a process dies before
its next kernel syscall; neither does today's option (c).

## 5. Observation, verifier, ACK and readiness

### 5.1 Exact meanings

The proposal adds a nullable `intersection` object to `ObservationReport` and
the block token `generation_intersection`. This token means **full desired
service is blocked**, while a proven subset can still forward. All other block
tokens keep their present meaning. An intersection report must carry the token;
any other blocking condition closes S and reports baseline/quarantine instead.

| Field during verified I | Required value/meaning |
|---|---|
| `applied_config_generation` | g, the last completed ordinary Apply/ACK; never h. No `MarkApplied(h)` for S. Before any full serving g there is no eligible S. |
| `applied_binding_revision`, `applied_policy_refs` | From the retained g checkpoint, never copied from current N as `report()` currently does. New desired receipt is represented only by `intersection.target_*`. |
| `member_forwarding_enabled` | true iff the nonempty S has current authorization and complete live proof. It means some member forwarding is enabled, not all h bindings. False in K/A until the respective proof/checkpoint permits a truthful serving claim. |
| `forwarding_block_reasons` | `[generation_intersection]` for healthy I. Not `policy_missing` or `lease_expired`: O/L and restrictive N are present and in force. On failure, false forwarding plus actual existing reasons; do not leave this token pretending I still holds. |
| `lease` object | Original L ID and its signed node expiry, unmodified. The earliest effective subset cutoff is separately in `intersection`; signed expiry alone is never readiness. |
| `lease` check | failed / `STALE_GENERATION`, detail `Full desired generation is awaiting a matching serving lease; verified intersection only.` This is deliberately a **full-generation** check. The subset authorization proof is `intersection.verified`, not a falsely passed ordinary lease check. |
| `qos`, `source_binding` checks | passed only after actual S read-back. Both are explicitly scoped by the sibling `intersection` object and its hash. QoS detail records actual read time, source and subset hash, not `candidate=h verified=true` as a full Apply. Failure closes S, retains actual error and read time. |
| `errors[]` | One `STALE_GENERATION`, `stage=lease`, fixed full-generation message above. Real conntrack pending error may additionally use existing `ISOLATION_POLICY_INVALID` / `stage=apply`. Failed candidate/read/containment rows keep their actual code/stage/time. No new error code, no invented verifier read. |
| `safety_status`, provisioning, data health | Baseline safety can remain passed, provisioning applying while h is pending; data health is the actual probe result. None proves full convergence or publication completion. |
| `dns_policy_evidence` | Retained actually applied content; content change is ineligible for S. Never label N's unapplied resolver document as applied. |
| `intersection` | Non-null only for a retained actual subset verification attempt. `verified=true` only in I with complete proof. Failed read retains false and actual time/error; baseline report can retain this failed evidence, but forwarding=false and token absent. No candidate object before any read. |

During P/K, do not reuse the old serving=true report as evidence of S. During
full A, report false until proof, even though release of the authorized full
guard may allow packets before the subsequent heartbeat. Record actual full
candidate/read evidence separately. On B, existing baseline ACK behavior can
advance to h; its lease mismatch remains the follow-up-32 stale-lease hold.
On X, suppress applied ACK/references as today, forwarding=false, identity
reason and freeze evidence; old S is never live readiness.

Define local `intersectionReady` as: current process created the transition;
same boot and original trusted L anchor; immutable O/L/A/N provenance matches
the intent; target still h; nonempty exact S; now is before every retained
deadline; no freeze/update/safety/storage/meter/cleanup-bound failure; complete
recent nft/tc/path verification of S including unchanged rates/ceilings and
timeout caps; active durable checkpoint and successful heartbeat write.
Recheck deadlines after the read. It is not `desired=applied=lease`.
The pre-heartbeat proof authorizes writing the first I heartbeat under the
same node lock as E/read-back/checkpoint; only successful atomic heartbeat
replacement completes readiness. This avoids a circular readiness guard and
prevents the watchdog reading a new subset against an old full-model heartbeat
between those stages. Watchdog lock/budget exhaustion still fails closed.
`fullReady` retains that equality plus today's other gates and requires no
active transition. Updater health/installation success and central publication
completion use **fullReady**, not `intersectionReady`.

The initial profile uses a **v2 local watchdog heartbeat** for intersections:
boot/mono/wall, serving=true, baseline_ok=true, generation=g (last full ACK),
reason=generation_intersection, policy_expires_mono_ns=min D(b), plus required
transition_id, target_generation=h and subset_sha256. Do not hide authorization
in the optional diagnostics `evidence` string. The updated watchdog verifies
the subset marker/identity rather than treating g's marker as a full model;
it never renews timeouts. Unknown v2 or missing fields fails closed; current
v1 readers reject v2 at their first line. K and A have no serving=true heartbeat
until their proof. Restore an ordinary full heartbeat only after the full
checkpoint; mixed agent/watchdog rollout is disabled. A failed heartbeat write
withdraws instead of claiming durable readiness.

Observation coalescing includes mode, transition ID, g/h, old/new/subset hashes,
binding count, verifier outcome and actual error stage/code, as well as existing
containment and alarm state. Read times alone do not manufacture a new
transition. The first failed S read and a later failed full h candidate are
distinct even when their generation and error code coincide. Retry preserves
the original report ID, body, anchor and read time byte for byte.

### 5.2 Proposed OpenAPI patch (illustrative diff, not applied)

This is the complete proposed new data shape. Runtime validators must enforce
the cross-field predicates above, which OpenAPI 3.0.3 cannot express fully.
`IntersectionU64` has the same decimal-string range rule as existing fields;
`IntersectionTime` has the same UTC Z rule. Signed desired JSON Schema and
generated Go/TS must be changed by contracts in the later implementation task,
including hashes/golden vectors. **No contract file changes here.**

```diff
--- a/contracts/openapi.yaml
+++ b/contracts/openapi.yaml
@@ components.schemas.DesiredState.properties
+        intersection_constraints:
+          allOf:
+            - $ref: '#/components/schemas/IntersectionConstraints'
+          description: Optional signed restriction witness. Absent means intersection unsupported; never grants admission.
@@ components.schemas.ForwardingBlockReason
-        Why member forwarding is off on the node (fail-closed reasons).
+        Why full desired member service is blocked. generation_intersection
+        permits only the explicitly verified subset; all other reasons keep
+        their node-wide fail-closed meaning.
-      enum: [not_published, identity_invalid, provisioning_incomplete, policy_missing, policy_expired, lease_expired, awaiting_member_grants, clock_untrusted, safety_check_failed, meter_path_invalid, meter_storage_failed, outbox_full, disk_low, release_not_supported, update_in_progress, recovery_required]
+      enum: [not_published, identity_invalid, provisioning_incomplete, policy_missing, policy_expired, lease_expired, awaiting_member_grants, clock_untrusted, safety_check_failed, meter_path_invalid, meter_storage_failed, outbox_full, disk_low, release_not_supported, update_in_progress, recovery_required, generation_intersection]
@@ components.schemas.ObservationReport.description
+        A verified intersection is partial service, not an ACK of target_generation.
+        Its applied generation/references remain the last full checkpoint.
+        lease fails with STALE_GENERATION; qos/source_binding are scoped to
+        intersection.subset_sha256. Ordinary generation mismatches retain
+        policy_missing/STALE_GENERATION, including baseline ACK holds.
@@ components.schemas.ObservationReport.properties
+        intersection:
+          type: object
+          allOf:
+            - $ref: '#/components/schemas/IntersectionEvidence'
+          nullable: true
+          x-omitempty: false
+          x-omitzero: true
+          description: Actual subset verifier evidence; absence/null never establishes subset readiness.
@@ components.schemas.TCReadbackEvidence.description
+        For an intersection read, candidate_generation is target_generation,
+        and a sibling ObservationReport.intersection MUST identify the subset
+        and the identical verifier completion time/outcome. This is not a
+        full-generation Apply. The D-106 sha256 projection is unchanged.
@@ components.schemas
+    IntersectionU64:
+      type: string
+      pattern: '^(0|[1-9][0-9]*)$'
+      maxLength: 20
+      x-go-type: decstr.U64
+      x-go-type-import: {path: github.com/TED-H-com/Freedom-VPN/pkg/decstr}
+      description: Exact uint64, maximum 18446744073709551615; checked without floating point.
+    IntersectionHash:
+      type: string
+      pattern: '^[0-9a-f]{64}$'
+    IntersectionTime:
+      type: string
+      format: date-time
+      pattern: '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$'
+    IntersectionConstraints:
+      type: object
+      additionalProperties: false
+      required: [version, members]
+      properties:
+        version: {type: integer, enum: [1]}
+        members:
+          type: array
+          maxItems: 4096
+          description: Complete eligible-member restriction snapshot; unique member_id. Empty denies all; missing document disables intersections.
+          items:
+            type: object
+            additionalProperties: false
+            required: [member_id, entitlement_revision, effective_service_end]
+            properties:
+              member_id: {type: string, format: uuid}
+              entitlement_revision: {$ref: '#/components/schemas/IntersectionU64'}
+              effective_service_end: {$ref: '#/components/schemas/IntersectionTime'}
+    IntersectionEvidence:
+      type: object
+      additionalProperties: false
+      required: [version, transition_id, base_generation, target_generation,
+        old_desired_sha256, new_desired_sha256, lease_id, lease_generation,
+        lease_sha256, original_request_mono_ns, subset_sha256,
+        earliest_deadline_mono_ns, bindings_count, read_at, verified, error_code]
+      properties:
+        version: {type: integer, enum: [1]}
+        transition_id: {type: string, format: uuid}
+        base_generation: {$ref: '#/components/schemas/IntersectionU64'}
+        target_generation: {$ref: '#/components/schemas/IntersectionU64'}
+        old_desired_sha256: {$ref: '#/components/schemas/IntersectionHash'}
+        new_desired_sha256: {$ref: '#/components/schemas/IntersectionHash'}
+        lease_id: {type: string, format: uuid}
+        lease_generation: {$ref: '#/components/schemas/IntersectionU64'}
+        lease_sha256: {$ref: '#/components/schemas/IntersectionHash'}
+        original_request_mono_ns: {$ref: '#/components/schemas/IntersectionU64'}
+        subset_sha256: {$ref: '#/components/schemas/IntersectionHash'}
+        earliest_deadline_mono_ns: {$ref: '#/components/schemas/IntersectionU64'}
+        bindings_count: {type: integer, minimum: 0, maximum: 4096}
+        read_at: {$ref: '#/components/schemas/IntersectionTime'}
+        verified: {type: boolean}
+        error_code:
+          type: string
+          allOf: [{$ref: '#/components/schemas/ErrorCode'}]
+          nullable: true
```

The 4096 limits are proposed **protocol bounds**, not an entitlement/device
quota or an invitation to increase existing document/body limits. The lower
existing document bound wins; oversized/unknown content uses baseline. Schema
versioning/compatibility needs the approval in Q1/Q2, not a silent version-1
extension accepted by an old strict signed-document parser.

### 5.3 D-106 and central consumers

D-106's existing TC digest remains exactly JCS of
`{reads, objects, truncated, tc}` with original array order and optional member
presence. Do not put generation, transition, subset hash, command timing or
lease anchor into that digest, nor reinterpret old digests. The **new** subset
hash has the distinct §4 domain and covers model semantics, not TC diagnostic
rows. A matching TC digest alone cannot prove an intersection or its ceilings.
The complete verifier must bind nft S marker, deadline/readback, actual tc
ceilings/classes, path and meter to the immutable S expectation. Retain actual
read time/outcome; `tc_evidence.candidate_generation=h` is scoped only by the
matching intersection object. On a later full h attempt omit intersection for
that read; keep the earlier rejected S observation immutable in the outbox.
D-106 owner must approve this context-dependent candidate meaning and golden
positive/negative pairs before any implementation.

Central classification is `generation_intersection` (partial service, full
convergence pending), never “applied h”, ready-for-publication, or completed
operation. Require authenticated node/incarnation, g/h ordering, matching
issuer lease ID/hash/generation, desired hashes, nonzero count, matching actual
read context, and a fresh accepted report. Monotonic values are boot-local:
central cannot compare them to its wall clock; it uses receipt freshness plus
the signed document expiry and never independently extends authorization.
The issuer's stored expiry clamp/tombstone and current identity/publication
state take precedence over L's original signed expiry, as in D-088.
Central classification is node-reported proof, not independent packet evidence.
Per-member AT acceptance also needs the local exact subset and packet joins;
a count/hash alone does not prove a particular member is admitted.

Separate preservation of an already published path from permission to create
or complete a publication: valid I may retain the existing route needed by S,
but cannot add a route, expand peer grants, or complete the h operation.
`PublicationDataPathFailure` must return the specific partial-service cause
before its ordinary applied<desired early return; it must never return success
for I. `ServingReadiness.IsServing` remains the full-convergence predicate;
callers that preserve existing routing need an explicitly validated partial
predicate. No healthy partial report may mask a real revocation, freeze,
publication withdrawal or isolation failure. Wire-nodes must test these
different consumers, not just rename a reason in the UI.

Wire-nodes follow-up 34's stale-lease hold remains distinct: baseline reports
with `policy_missing` plus lease-check **or lease-stage-error**
`STALE_GENERATION`, even when applied==desired. Classify real freeze/isolation/
storage failures first; then valid intersection; then that specific hold before
generic check_failed/forwarding_disabled. Do not classify using
STALE_GENERATION alone. A failed intersection read is a safety failure, not
partial service. Persistent generation incidents span I→B, supersession and
process restarts. Mirror follow-up 32's three-Step-or-30s local diagnostic alarm
from first transition intent, and keep persistent duration centrally; do not
reset duration on replay or use the applied<desired one-shot grace as its timer.
Matching h lease alone does not end the partial-service incident; full proof does.

**Old central fails safe:** rollout must keep this feature off until the
compiler, observation validator/classifier, D-106 consumer and watchdog are
upgraded together. Signed constraint presence is necessary, not sufficient
capability negotiation. An old strict schema rejects the extra object/token;
on that unsupported-report response the node abandons I, baselines and never
strips fields to disguise it as a legacy serving report. If an old permissive
consumer ignores extensions, g<h and the failed lease check prevent full
readiness/publication under existing gates. Never depend on that behavior as
deployment negotiation; a consumer using the forwarding bit alone must be
fixed before enablement. Transport outage after an already approved rollout
does not revoke S by itself; old anchored limits and all local gates still do.

## 6. Requirements to netlab / node/enforce (not an enforcement design)

1. **Inputs:** immutable O/L/A/N hashes and normalized S; exact previously
   admitted identity/revision/address tuples; original deadlines and observed
   kernel expiration caps; unchanged tc rates/ceilings/assignments and host
   reserve; topology/meter identities; counter map; freeze epoch; pending
   revocations and persisted attempt count. Reject stale/ambiguous inputs.
2. **Ordering:** provide a bounded revocation-only atomic nft transaction at E.
   In both directions, removed/changed authorization and newly denied
   destinations/resolvers must be rejected **before established/related
   accepts**, accounting or any bypass. Do not wait for tc, probes, lease
   fetch, counter sampling or conntrack deletion before removal. Preserve
   control access and `inet fvpn` ownership. No host route/DNS changes.
3. **Timeouts and marker:** E retains `fvpn_generation=g` and atomically adds
   mode=intersection, transition ID, target h and subset hash markers. An
   ordinary full-g verifier must reject that mode. Full h's guarded install
   removes those markers and installs its ordinary h marker; baseline clears
   them with member admission. Never make `fvpn_generation=h` alone look like
   a completed h Apply. Preserve
   kernel timeouts without extension, including late/uncertain commits and
   process death. Existing reserve/granularity/guard bounds are unchanged. If
   an nft plan cannot safely shrink timeout and policy at E without a broader
   outage, baseline. Netlab must establish the ordering and commit primitive;
   this document does not prescribe its chain/set layout.
4. **Accounting:** retain named device counters, unchanged member class minors,
   sampler ownership and ledger epoch through successful E. Do not call the
   current baseline-based `handoff` for successful I. Retain retired identity
   attribution until final sample is durably recorded. Admission counts upload
   and download exactly once; drops remain uncharged. No reset, double sample,
   synthetic zero, counter rename or address reuse can erase old bytes. Baseline
   fallback still performs today's final handoff. Global meter ambiguity closes.
5. **Conntrack:** after removal, attempt exact IPv4 source **and** destination
   deletes for every removed/rebound original host identity in the owned
   datapath context. Preserve unrelated sibling flows; never flush conntrack.
   Destination-policy restrictions must apply to established flows at E even
   before cleanup. If deleting only affected flow tuples cannot be proven for
   such a policy change, choose baseline, not a broad sibling flush. Keep exact
   unsuccessful addresses/identities and distinguish verified zero matches
   from command failure. Do not silently drop pending deletes on generation
   change or re-admission. Block reuse until cleanup succeeds.
6. **Bounded exception:** only after complete S verifier success may transient
   exact-delete failure leave S serving. Preserve the current **12 failed
   attempts** bound; at most one retry batch per Step, persist an attempt
   reservation before the batch, do not reset it on transition/restart, and
   baseline on exhaustion. A missing binary fails immediately. Twelve is an
   attempt count, **not** a two-minute wall-clock guarantee. Replay/Retry-After
   cannot extend D. Retry success clears only successful tuples; full recovery
   must not admit reused addresses with unresolved old flows.
7. **Read-back:** verify the full S model, generation/transition/hash markers,
   admission-before-established order, absence of bypass, both-direction
   source bindings, destination/DNS restrictions, original absolute timeouts,
   accounting hooks, exact tc parent/member ceilings/rates and class identities,
   and unchanged member path. Retained inactive classes/counters must be
   explicitly owned and non-admitting, not mistaken for foreign objects or
   live members. Generic TC shape summaries alone are insufficient.
8. **Failure:** mutation, read, deadline, persist, meter or topology failure
   selects baseline; foreign attachment/ownership/bypass selects quarantine
   as today. Retain failed candidate and actual read separately. Do not restore
   O or S after full Apply failure. Failed baseline/containment stays visibly
   failed with watchdog/kernel deadlines intact.

The existing `node/enforce/apply.go` loads state and tc, renders, performs
ownership reads and **TCBatch before NftBatch**. Reusing it would delay E behind
shaping and could serve old grants after a known revocation. Its generic
generation verifier, baseline rollback and guard handling also do not identify
S. Full Apply remains appropriate only after matching current authorization in
A; the revocation boundary needs a separate operation and verifier contract.
Even at A, ordinary shape-first Apply must start behind the explicit baseline:
otherwise a looser h class could overwrite S's stricter class while S still
uses L. An optimized full promotion without that baseline is **not** part of
this first profile. Consequently it preserves the sibling during the lease
wait, but still has a full-Apply outage at promotion. D-133 must establish
whether the benefit survives this cost; this note does not promise AT02
uninterrupted forwarding or an improved p99 on fast-renewal workloads.

## 7. Executable coverage and later test plan

The test-only reference takes authenticated, normalized abstract inputs, not
real envelopes, host configuration or an enforcement executor. It cannot be
used to claim kernel/issuer verification. Its exhaustive tables cover identity
equivalence classes; all tighter/equal/looser combinations of the four ceilings;
each deadline as the deciding minimum; boundary, reserve, overflow and replay
times; and semantic CIDR/denial/resolver intersections. `Baseline=true` always
means zero authorization even if a mathematical candidate is retained for
inspection. Run `go test -count=1 ./node/agent -run '^TestF33_'`.

Future engine/unit/fault tests must exercise the actual signed inputs and all
state transitions; the reference is the independent expected subset, not the
production implementation copied into its own assertion:

| Negative / normal matrix | Required assertion |
|---|---|
| Delayed, replayed, lower/wrong-generation, equivocal and empty leases | No new anchor/grant; no changed old lease. Wrong response cannot promote A. Valid empty current lease closes S immediately; stale empty/unverifiable response never becomes permission. |
| Exhausted control/local/read budgets, Retry-After and outbox full | E never waits on network; original deadlines persist. Shared local budget does not stack. Full outbox/storage failure closes. Repeated immutable observation bodies and alarm transitions survive retry. |
| Desired supersession before/after E and before full guard release | Baseline, no chained subset; target hash/epoch checked again; no stale ACK or resurrected device. |
| Expiry and freeze at every P/K/I/A boundary and during read/write | No packet past cutoff; freeze wins next mutation; no active/full checkpoint published after freeze. Deadlines before/after read; original reserve and watchdog behavior. |
| Address/member/device/peer reuse; changed revisions; additions | Never admit from account ID intersection. Duplicate/unknown global identity closes. Old counter attribution and exact conntrack backlog survive reuse. |
| Crash at every row in §4, including in-flight freeze response | Baseline before any adoption; no transient heartbeat or lease cache can restart S. No acknowledgement without required durable checkpoint. |
| Same-boot graceful stop, SIGKILL, reboot, clock anomaly/VM restore | Feature-enabled Boot never adopts, full reacquisition required; option (c)'s existing completed-F restart behavior remains separately covered. |
| Failed Apply/live read/path, foreign tc, baseline failure | D-135 no restoration; rejected subset/full candidate remains visible with actual error. No fabricated successful read. |
| Partial/all conntrack failures through attempt 12 | Revoked established TCP/UDP denied immediately; sibling retained only with verified S and bound remaining; restart cannot reset count. |
| Protocol/digest/central compatibility | Old D-106 vectors byte-identical; S and full h reads cannot be confused. Old schema rejects, newer central classifies partial, publication/update readiness remains false. |

Netlab must record these lab rows, with independent ingress/egress and both
directions, not only agent boolean assertions:

| Row | Setup and evidence |
|---|---|
| N1 revocation sibling | Established TCP/UDP for revoked A and unchanged B, plus never-admitted C. Continuous B stream across E, lease wait and full recovery; A/C cannot forward after E. |
| N2 identity reuse | Same address reassigned to another device/member/peer, plus pending delete. No stale session reuse or counter misattribution. |
| N3 policy restriction | Denial union, overlapping CIDRs, exact DNS exception removal, IPv6 denial. Established flows obey restrictions before accepts. Unsupported resolver-content/topology case baselines. |
| N4 QoS | Read and saturate upload/download for every retained member and node. Both stricter limits and paired-A ceilings hold. Tighter tc requirement takes baseline at E; no optimistic partial shaping. |
| N5 timers | Delay render/commit/read, kill agent/watchdog at each relevant boundary; remaining kernel expiration never increases, reserve/granularity unchanged, packet cutoff ≤D. |
| N6 fault/cleanup | Partial nft/tc/path read, foreign filter, conntrack failures 1–12, missing binary, disk failure and supersession/freeze. Baseline/quarantine with first rejected evidence; sibling cleanup is exact. |
| N7 meter/restart | Bytes across E, death and final sample; no lost/doubled attribution, epoch/period changes baseline, same-boot S restart and reboot both reacquire. |
| N8 observations | Join accepted central report ID/boot/transition/subset hash to actual producer read/heartbeat and packets. Replayed/backlogged reports cannot qualify a newer window. |

D-133 A/B uses the [L-2-corrected rule](followup31-generation-change-availability-2026-11-01.md#recommendation-and-d-133-acceptance-plan)
and **D-138 judging**, not a new tolerance. Freeze hashes for A=current option
(c) and B=approved implementation; run A1,B1,A2,B2 on the same host/kernel,
fixture/input stream, observation mode, Go settings and probe cadence. Each leg
includes the full agent and live suites and **at least 40 restrictive and 40
non-restrictive changes**. Restrictive rows include a continuously entitled
sibling, revoked source and attempted addition. Report how many transitions
qualified for S versus fell back, including QoS/revision/contract causes.

Require no competing suite and six interval samples from `vmstat 5 7` with
r≤3/id≥80 before startup, before traffic and after; exclude its since-boot row.
Retain every rejected preflight, failed leg and unqualified interval/run. Do not
replace failures with later passes or compare different instrumentation.
Record publication/receipt, E start/end, lease requests/acceptance, every
complete/incomplete guard, Apply, actual live/path verification, heartbeat,
independent first/last egress and every loss inside/outside guards, using original
run/boot/sequence joins. No E command-end timestamp is called exact packet cutoff.

Publish full vectors and nearest-rank p50/p90/p99/max plus censoring/failure
counts. Both B legs require no later revoked-source cutoff than paired A's
permitted cutoff and old D, no new/stale/additional binding, no increased expiry
bound, no unmatched safety failure, non-worse four reported outage/loss
percentiles and strictly lower restrictive p90/p99. Compare rendered **and
read-back upload and download ceilings for every survivor and the node** against
paired A and the stricter authenticated O/N ceilings. Any higher or missing
comparison fails even if all loss percentiles improve. Keep fault populations
separate and count every loss; guard-only percentiles do not establish success.
Any live timing failure remains unresolved pending D-133 attribution; no
threshold increase. Cross-build amd64/arm64, then record physical-arm64 results
separately. No lab/performance or real-arm64 result is claimed by this design.

## 8. Numbered decisions still required

These are approval questions for the recorded post-r19 decision, not choices
left for individual implementers to improvise. If rejected, keep (c) for that
item and deliver the remaining work.

1. **Cross-generation restriction authority and entitlement witness. Owner:
   orchestrator/product owner + contracts + wire-nodes issuer/compiler.**
   Recommend approve only the §3 exact-revision rule and signed
   `intersection_constraints` snapshot, with versioned signed-schema rollout.
   Current desired documents cannot prove it. If the authoritative compiler
   cannot emit consistent per-member witnesses without inventing entitlement,
   reject (b) until it can; no “member IDs are enough” substitute.
2. **Observation/ACK/D-106 representation. Owner: contracts + D-106/netlab
   owner + wire-nodes, recorded by orchestrator.** Recommend §5: g ACK, explicit
   partial-service token/object, failed full-generation lease check, scoped
   subset verifier, unchanged TC digest and separate model hash. Require
   paired compatibility/golden vectors and versioned watchdog support. Reject
   full h ACK or a passed ordinary lease check while L still names g.
3. **Revocation-only feasibility and no-later E. Owner: netlab, reviewed by
   r19 and orchestrator.** Recommend no tc mutation at E, bounded nft-only
   restriction with exact counter/conntrack preservation and timeout caps.
   If this cannot retain the sibling without delaying E, baseline for that
   input. Do not relax §8.2 to buy availability.
4. **Fallback frequency versus complexity. Owner: orchestrator + AT members,
   isolation and QoS lanes.** Recommend measure eligibility separately in
   A/B. If real compiler revision churn, DNS/QoS changes or required restart
   work make most revocations fall back, or the no-adoption restart cost is
   unacceptable, do not ship this complexity merely
   because the pure-revocation unit case works. No cost-effectiveness claim
   yet; unchanged sibling revisions must be demonstrated on native AT inputs.
5. **Activation/rollback and partial-service operations. Owner: contracts,
   wire-nodes, installer/updater and orchestrator.** Recommend staged deployment
   with a disabled-by-default explicit node setting and central rollout
   allowlist only after all readers support the signed schema, observation and
   heartbeat versions. No autonomous capability inference. Central retains
   incidents across boots; publication/update completion requires full proof.
   Rollback disables the feature and baselines before old binaries run.

## 9. Dated lane hand-offs — 2026-11-01

* **Contracts / D-106:** approve or reject §5 schema, signed witness and
  contextual verifier semantics; generate OpenAPI/JSON Schema/Go/TS and
  compatibility/golden vectors only after the recorded decision.
* **Wire-nodes:** atomic compiler witness snapshot, issuance/readiness tests
  with ACK g (no circular requirement for ACK h), observation validation and
  persistent partial-service classification alongside follow-up 34's baseline
  stale-lease hold. Report UI/operation partial service without completing h.
* **Netlab:** design and prove §6 revocation-only operation, subset verifier,
  original-timeout preservation, counters and 12-attempt cleanup. Deliver N1–N8
  with first-failure evidence. This lane has not edited `node/enforce`.
* **AT lanes:** members checks unchanged sibling plus revoked/additional/reused
  sources; isolation checks established flows/freeze/crashes; usage checks
  continuity; QoS checks all ceilings; handover checks truthful contract and
  evidence joins. D-133/D-138 trial only after implementation; no loss waiver.
* **Agent-core / watchdog after approval:** separate transition type/store
  schema and mutation path; no use of recovery rewrite; mode-aware verification,
  report/outbox keys and v2 heartbeat; baseline-only transition restart. Carry
  unchanged updater/installer consumer gates and coordinate installer rollout.

Until Q1–Q3 receive one recorded approval after r19, implementation is blocked
for those items, and (c) stays in production. The reference and design remain
reviewable even if the final decision is to reject (b).
