# Revoke one device and keep its sibling serving

This bounded AT02/§8/§10 case starts with two native NetBird devices belonging
to one real member. Both must have product-authorized device projections,
signed desired-state bindings, an explicitly selected published exit, controlled
origin receipts and increasing named admission counters.

The runner issues `DELETE /api/v1/member/devices/{id}` with the member session
and an idempotency key. HTTP 202 starts the check. It waits at most 90 seconds
for removal from signed bindings and a newer serving generation, then reads the
central node projection. No agent restart, state clearing, group repair or
projector SQL mutation is permitted.

The negative request must reach node `fvmb0`, increment the unknown-source
drop counter `ctunku`, leave the revoked device's `u_`/`d_` bytes frozen, and
produce no receipt at the authenticated controlled-origin slot. Its conntrack
entries must be gone. A second counted request from the other device must still
succeed, member accept rules must remain, and the agent PID must be unchanged.
The executable check uses the image's packaged `conntrack`; there is no runtime
package installation or enforcement bypass.

After revocation, the runner admits a replacement native device for the same
member and requires a counted origin request. The final agent PID and member
accept rules are checked again after that replacement applies.

`delete_to_first_observed_drop_ms` is the timestamp of the first captured
post-apply dropped packet minus DELETE start. It includes polling and capture
setup, is an observation only, and is not an exact stop-latency SLA. The record
retains route, capture, rule, origin and generation evidence so a projection,
route-withdrawal or enforcement failure can be assigned to its actual owner.

In all three follow-up 10 runs, NetBird removed the revoked client’s advertised
exit before the post-apply request reached the node. Sibling and replacement
traffic succeeded, the node stayed ready, and revoked counters froze with no
conntrack or origin receipt. The strict ingress/drop assertion remains blocked,
and no DELETE-to-node-drop timing was observed. Run 3 also saw two aggregate
unknown-source drops during the transition; their source and first-packet time
were not captured.

See [the current record](../runs/2026-10-30/device-revoke.md) and
[follow-up 10 verification](../followup10-verification.md).
