# Device authorization fences — 2026-10-31

Follow-up 31 closes r16 H-1 for D-077 (§9.4, §16.3, §17.4; MEM03, ENT01,
ENT02). A row in `device_authorization_fences` is an open fence. There is no
expiry or separate closed status: explicit re-authorization deletes it, and
ordinary revocation also removes it while changing the device status.

The desired-state binding reader treats an open fence as revoked. The lease
entitlement reader excludes that device from its device list, including its
locked eligibility read. Both use the same row-existence predicate as the
public device projection. The existing `device_id` primary key supports these
lookups; no migration, setting, API field or dependency is added.

Signed serving leases remain member-wide. A member can retain a lease for an
unfenced sibling device. Device admission requires both a valid member lease
and the device's signed binding, so removing that binding denies the fenced
peer without withdrawing its sibling's entitlement. This follows the existing
revoked-device behavior; it does not introduce device-level lease documents.

## Existing grants and timing

Ordinary revocation already follows `RevokeDevice` → durable `device.revoked`
job → `SyncDevice` → `RecompileBatch`. The compiler replaces each affected
current signed envelope and advances the desired generation. Node observation
remains separate evidence; compilation does not fabricate an applied ACK.

Fences open inside `SyncDevice`, including calls from `SyncMember`. The worker
now checks for current-incarnation envelopes still binding any fenced device
of that member after each of its three sync call sites. It passes those node
IDs through the existing device-revocation compiler path before acknowledging
the detecting job. This covers `netbird.device_sync` and `members.reconcile`,
which previously returned without compilation, as well as the ordinary member
and device intent jobs. An exit in a removed group is still selected if its
stored envelope retains the binding; an unrelated exit is not selected.

The check also runs when sync returns an error: a fence may already have
committed before an identity-mapping error or another device's sync failure.
Compilation failure leaves the original job retryable. A retry finds the
persisted fence and remaining signed binding rather than relying on a transient
“new fence” flag. Once the bindings are removed, repeated observation jobs do
not advance generations. A concurrent explicit authorization is read by the
compiler normally; the helper never changes device status or opens a revoke
operation.

The compiler holds the existing shared member/entitlement locks until its
transaction commits. Fence creation already takes the member update lock. This
serializes a first or previously grantless compile with fencing: either the
compile commits first and its binding is visible to fanout, or the fence commits
first and the binding read excludes it. A compile paused at signing cannot
create an unnoticed grant after the detecting job is acknowledged.

This is the same asynchronous propagation boundary as ordinary revocation.
An already delivered signature is not remotely erased. The agent must fetch
and apply the new generation; offline nodes retain the existing policy/lease
deadlines (§9.4, §16.3). Restore holds and compiler failures retain their existing
retry behavior. No real-network convergence latency is claimed.

## Adopted review proofs

The five files below were copied unchanged from
`origin/r16-rev13-d089-d099-review`, commit
`e672d5d3e36f060ab4600fd8507f4c12e105be54` (`Agent: grok/grok-4.7`), reviewed
against base `5575470`. Original test names, comments and `TODO(review)`
failure messages remain intact for provenance; they describe the review base.

| Adopted proof file | Result and scope |
| --- | --- |
| `internal/desiredstate/review_r16_fence_grant_integration_test.go` | H-1 reproduced on the follow-up base in both grant readers. After this fix the fenced and revoked devices are absent, and an unfenced sibling is still bound and listed for leasing. |
| `internal/desiredstate/review_r16_hold_integration_test.go` | D-095 receipt cutoffs, current-incarnation isolation and device-revocation writes remain intact. Passed before the fix. |
| `internal/nodes/httpwire/review_r16_d099_test.go` | D-099 generation-only advance reaches the signer. Already passed with follow-up 29 / #277. |
| `internal/nodes/httpwire/review_r16_d099_integration_test.go` | Cached HTTP desired state advances beyond the restored node's applied generation. Already passed with #277. |
| `internal/nodes/httpwire/review_r16_replay_integration_test.go` | Keyed installer-result replay returns the same stored ACK. Already passed with #277. |

`TestD077_FenceRecompilesBoundExitsBeforeJobReceipt` adds worker evidence for
the detecting observation job, periodic member sweep, a committed fence followed
by compilation failure and retry, and a committed fence followed by lost identity
mapping. It verifies signed replacement on two exits, including a removed group;
no generation change on an unrelated exit; no fabricated applied evidence;
repeat sync stability; original-key replay retaining the fence; and a new
explicit authorization restoring the device list and signed binding.

`TestD077_FenceSerializesWithFirstBindingCompile` pauses a real compiler after
eligibility reads and runs the fence job concurrently. PostgreSQL lock evidence
establishes the ordering, and the final signed envelope must remove the device
before the job returns. It covers the no-prior-envelope race found during the
independent patch review.

These are central PostgreSQL, compiler and worker tests with generated signing
keys and test-only external peer read-back. They are not live NetBird, packet
forwarding or physical arm64 evidence. See the existing
[publication gate diagnostic note](2026-10-31-publication-gate-cause.md) for the
separate desired-versus-applied operation display.
