# Revoke one device and keep its sibling serving

This bounded AT02/§8/§10 case starts with two native NetBird devices belonging
to one real member. Both must have product-authorized device projections,
signed desired-state bindings, an explicitly selected published exit, controlled
origin receipts and increasing named admission counters.

For follow-up 11, `Stack.PauseNativeControl` sends SIGSTOP to the revoked
client's NetBird userspace service after its route and kernel WireGuard session
are established. A further counted origin request must succeed **while paused**
before DELETE. The exit's agent and watchdog keep running. This deliberately
stale client cannot consume the route withdrawal; no key, route, source binding
or product enforcement is fabricated. Cleanup always sends SIGCONT. If this
transport cannot stay live, the record names that missing lab primitive instead
of claiming an ingress drop.

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

## Follow-up 11: stale route retained, ingress primitive still missing

Run `76f0725c-2264-4021-a553-60f6c921e32b` confirms
`paused_client_counted=true` after SIGSTOP. DELETE returned 202 and the exit
projected generation 8, ready/passed, with the same agent PID. The stale client
still routed `192.0.3.10 dev fvnb0 table netbird`, but the post-apply node capture
reported `0 packets captured`; `ctunku` stayed `0 → 0`. Revoked admission bytes
froze at `5262 → 5262`, conntrack was empty and origin receipt zero. Sibling
and replacement flows were counted; SIGCONT cleanup succeeded.

A later read-only NetBird inventory during that same run reported
`usesKernelInterface=true` and no peer for the revoked address at the exit.
This supports upstream transport withdrawal; it does not establish the precise
removal time. Keeping the client's kernel route alone cannot guarantee packets
arrive at the exit after the remote peer disappears. No node-rule defect is
inferred from zero node ingress. DELETE-to-first-node-drop is **not observed**.

Missing primitive: a controlled tunnel-ingress traffic path that remains usable
at the node after NetBird removes the revoked peer, while the real agent keeps
applying signed state. It must attribute post-DELETE packets to the old device
source and measure `fvmb0` ingress plus named drop/admission counters, without
restoring a revoked peer/group/binding or altering product enforcement.
**Owner: w3-devstack-e2e with w1-netlab for the isolated ingress fixture.**
The current native pause hook supplies the stale client only; it does not supply
that independent node-side ingress path. This strict assertion remains blocked.
