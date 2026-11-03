# Follow-up 31: source-binding hook feasibility

2026-11-03. **Item 1 stopped at its explicit feasibility condition.** No
source-binding hook was added. The separate populated-hold source-address row
remains **not exercised**. There were **zero stack attempts**, zero injections
and zero retries; no node, incident, report, audit or injection/removal receipt
was produced by this follow-up. Central classification below is code analysis,
not a live observation or an expected result imposed on a future experiment.

Product source inspected: `63a30c412106f9f5bc6c1738cf23cd212634ce28`.
Members 17 source inspected read-only with `git show w4-at-members:<path>`:
`1019e9591c68a3d8f9323dd51259d94d1e5e5bf8`. Its `requests.md`,
`F17-populated-holds.md` retained-safety row, and `populated_hold_test.go`
distinguish the existing TC fault from the unexercised source-address row.

## What the node checks

`bindingOK` is a local variable in
[`Decide`](../../../../../node/agent/decide.go), not an independent live
source-address verifier in the engine. `Decide` calls `buildModel(in)` and:

- Starts `bindingOK=true`, `qosOK=false` (lines 208–210).
- Sets both false on any model construction error (lines 211–222).
- With a valid desired document and in-force matching lease, sets `qosOK=true`;
  if desired bindings exist but the model admits none, sets `leaseOK`,
  `bindingOK` and `qosOK` false (lines 223–241).
- `checks()` reports `source_binding = bindingOK && leaseOK` and
  `qos = qosOK && leaseOK` (lines 332–355). A retained Apply error or missing
  dependency overrides **lease, qos and source_binding together** with the
  same failure code (lines 357–368).

`buildModel` (line 586 onward) intersects the signed desired device bindings
with the signed lease's member IDs; checks binding validity windows and
monotonic member/node/desired/offline deadlines; accepts nonempty IPv4 host
addresses (bare or /32), requires a nonempty peer ID, and rejects duplicate
admitted addresses. It also builds rate policy and validates capacity and
resolver addresses. Its failures share the model error path. It does not
compare the peer ID with live WireGuard AllowedIPs or compare a member's source
address with an interface address. Removing one eligible binding while a
sibling remains would not by itself make its aggregate admission predicate
false.

Thus, for an evaluated serving check, **a failed source_binding entails a
failed qos** in this implementation. A generation hold already sets
`leaseOK=false`, so source_binding cannot pass again while that mismatch
remains, even after a hypothetical address fault is removed.
[`attachServingRead`](../../../../../node/agent/serving_evidence.go) adds
read-back detail to qos but explicitly preserves Decide's status/error; it
does not provide a separate source-binding result.

## Why namespace faults do not meet this request

| Candidate | Actual path | Reason it cannot qualify |
|---|---|---|
| Alter an nft source match / allowed element | `enforce.VerifyServing` reads nft and TC; a serving verification failure reaches `rejectServing` → `noteApply` | Shared Apply error fails qos and source_binding together. During an ordinary hold, the node installs a baseline; `auditBlocked` promotes `TC_FOREIGN_FILTER` and retains quarantine, not arbitrary nft source-rule errors. It cannot turn an nft mutation into an independently reported source-binding cause in that hold. |
| Remove a local member-interface address or member route | `SystemPlane.MemberPath` checks interface UP, an IPv4 address, retained topology hashes and route-to-peer interface | This is member-path topology verification, not `bindingOK`. `rejectMemberPath` calls `noteApply`, producing the same shared check override. The ordinary non-serving hold path installs a baseline and returns before these serving Apply checks. |
| Change peer-side source address / WireGuard AllowedIPs | No live peer-address read feeds `bindingOK` | Packet loss or a different meter/probe result is not evidence that this check failed for that cause. |
| Expire/revoke signed admission or alter capacity | Model/lease checks above; an actual API command can also advance generation | Conflates authorization/lease or rate-policy failure with a source-address fault; qos also fails. Editing/replaying/synthesizing signed documents is forbidden. |
| Existing clsact fault on `fvmb0` | TC verifier and retained Apply error | This is the members retained-safety row already exercised; it supplies no separate source-address evidence. |

Trace references:
[`verifyServing`](../../../../../node/enforce/verify_tc.go),
[`verifyServingRuleset`](../../../../../node/enforce/verify.go),
[`verifyCurrentServing`, `rejectServing`, `auditBlocked`](../../../../../node/agent/serving.go),
[`enforce`, `noteApply`](../../../../../node/agent/engine.go), and
[`MemberPath`, `rejectMemberPath`](../../../../../node/agent/memberpath.go).
In particular, `verifyCurrentServing` returns immediately when not serving;
the blocked audit's deliberate TC priority explains why the existing members
fault is reportable during the hold without establishing a source-address row.

## Central classification: read-only analysis

[`generationHoldSafetyFailure`](../../../../../internal/nodes/generation_hold.go)
ignores the known dependent `source_binding` failure only when its code is
`VALIDATION_FAILED` and detail is exactly `No member binding was admitted.`.
It similarly exempts the exact dependent qos sentence and lease
`STALE_GENERATION`. Other failed checks (including duplicate failures) or
independent stage errors return `check_failed`; remaining safety, backend,
internet-only, meter and block-reason conditions can return
`isolation_unconfirmed`. Exact report fields matter, including any retained qos
read-back detail; a check name alone cannot establish the classification.

[`PublicationDataPathFailure`](../../../../../internal/nodes/publication_evidence.go)
gives this safety result priority over `stale_generation`. Incident recording
in [`recordGenerationHoldTx`](../../../../../internal/nodes/generation_hold_store.go)
is separate: a hold marker can still produce samples, and a known matching
generation without that marker can close the incident. Incident closure is
not proof of restored serving. None of these outcomes was observed live here.

## Required owner work

2026-11-03 request to orchestrator / **w2-agent-core**: define and expose a
real source-binding verification result, with a specific cause and member/device
scope, separate from the generic Apply/qos result. It must run while a matching
lease is withheld, against an explicitly defined retained binding expectation,
and distinguish expected deny-all hold state from an independent source fault.
Specify what actual node state is checked and how its restoration is verified.
Do not create a test switch that merely forces a report field. The node must
keep control reporting and fail-closed forwarding through this verification.

The requested independence also needs a reporting contract: current
source_binding and qos are both lease-dependent, so a source-only pass/fail
transition during a continuing hold cannot be represented by these statuses.
Agent-core and the orchestrator must choose how the independent result is
represented, and route any contract/central change to its owners. This lane
cannot implement that in read-only `node/`, `internal/` or `contracts/`.

After that prerequisite, devstack can implement an opt-in node-role (and, if
applicable, member/device) hook with a hard maximum duration, explicit removal,
idempotent cleanup on the test/handle cleanup path, and verified state read-back.
Its retained record must carry injection/removal commands, exits, before/after
state and UTC bounds, including failure/timeout attempts. A future fresh stack
must join at least two populated lease members, hold arm, node/incarnation,
incident, accepted reports before/during/after, escalation audit, hook receipts
and observed central classification. Neither this document nor the existing
D-102 lease-delivery seam satisfies that future proof.

Coverage remains open for §8.2, §10.2 and §16.3 / ISO02, AT02 and AT13. This is
a feasibility finding about the requested test seam, not a claim that the
product's packet-level source isolation is absent or that those ATs pass.
