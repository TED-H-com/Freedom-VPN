# Follow-up 21: members 11 fixture attribution (TEST ONLY)

This is analysis of the original `7edb263f` run on harness `33abb2fd`, product
baseline `96ec0630`. No packet is replayed, no topology or health criterion is
changed, and neither failed/blocked acceptance row is replaced. The
[structured input and timing extract](../followup21-members.json) pins all six
source artifacts by SHA-256 and retains the original twelve generation-13
Apply records, their lease identities, and the two exact injection inputs.

## F11-01: the first unobserved segment

The [original capture](../at/members/F11-01-forwarding.json) proves inner IPv4
TCP egress at member `fvnb0`: five packets to the controlled origin's TCP/443,
client port 20001, in a 9,016 ms observation. The route read changes `eth0` to
`fvnb0` in table `netbird`; the daemon reports connected and kernel interface.
The node's 9,033 ms observation and origin's 9,094 ms observation see zero
matching packets. Both node counter reads succeed at generation 5, retain six
accept rules, and show zero device and diagnostic bytes, including `ctbase`.
There is no origin HTTP request or response.

**The first unobserved segment is after the member's inner tunnel egress and
before the node's observable inner ingress.** It is not an observed node
baseline drop or an origin response failure. The fixture cannot say where
inside that segment the packets stopped. Its AF_PACKET observer in
`tests/e2e/at/members/cmd/probe/traffic.go` filters inner IPv4 TCP by origin and
client/server ports; it does not retain encrypted WireGuard UDP, peer handshake
or routing-key state, transport endpoints, relay hops, or outer link drops.
`Connected` is daemon connectivity, not proof of a working peer data path.
The observer also has no socket-loss statistics or per-packet monotonic bounds.
A zero filtered count therefore is not proof that no outer packet reached the
node. Netlab 21 owns the missing transport-hop interpretation; the MTU captures
from other flows cannot be substituted for this uncaptured flow.

AT01 stays blocked. A later working connection cannot repair this sample.

## F11-03: what the failed health result actually asserts

`ObserveMemberIngress` is a measurement/injection helper, not a replacement
agent health check. It reads counters, starts two bounded UDP/9 captures and
runs one `fvpn-netlab inject` per source inside the node. The omitted mechanism
argument selects `tun-mirred` for `fvmb0` link type 65534. Its positive control
requires the admitted device's `u_<hex>` counter and uplink/origin receipt; its
negative control requires `ctunku` and no device/uplink/origin increment.
The original revoked source meets the negative criterion (+1 packet / 29 bytes).
The admitted sibling instead meets baseline (`ctbase` +1 / 29) and fails its
positive criterion. Injection returning success means injection succeeded,
not that forwarding or agent health passed.

The agent's separate `HEALTH_CHECK_FAILED`, `stage=apply`, `apply failed` is
`Engine.noteApply`'s sanitized fallback for a failed enforcement call. It is
not proof of a failed DNS/tunnel reachability challenge. The retained reports
still show tunnel probe passed, with `Egress probe used no extra target.` The
real enforcement path uses `enforce.System`, real nft/tc, a 100 ms guarded nft
install budget, guard release, then live nft/tc/timeout verification. A failed
install rolls back to baseline and does not reach the latter checks.

| Original Apply sequence | Render → command return (ms) | Install OK | Guard released | Verification records |
|---|---:|---|---|---:|
| 387 | 15.202709 | true | true | 1 |
| 395 | 23.710096 | true | true | 1 |
| 403 | 23.768703 | true | true | 1 |
| 411 | 238.823403 | false | false | 0 |
| 417 | 184.488440 | false | false | 0 |
| 423 | 292.382545 | false | false | 0 |
| 429 | 244.369512 | false | false | 0 |
| 435 | 268.957078 | false | false | 0 |
| 441 | 304.391743 | false | false | 0 |
| 447 | 272.238447 | false | false | 0 |
| 453 | 285.659003 | false | false | 0 |
| 459 | 13.222095 | true | true | 1 |

All records share the original run/boot and generation 13, one-member lease
`fab24c50-ca13-4380-958f-8d9dbb3112cd`, issued
`2026-10-31T18:31:04.625204232Z`, expiring `2026-11-01T18:31:04.625204232Z`,
request monotonic `55002138024706`, deadline `141402138024706` ns.
The structured extract preserves each original bracket rather than translating
wall time into an exact kernel commit. Sequence 453's rollback succeeds in
`55052609280751–55052943484892` ns; subsequent verification-failure baseline
calls and the later serving sequence 459 remain in the original capture.

The shared fixture supplies `fvmb0` / `fvup0`, host reserve 1,000,000 bit/s,
5-second poll/heartbeat, 30-second watchdog staleness / 5-second watchdog
interval, TEST ONLY mode and detached timing. It supplies no replacement Apply
health callback and no relaxed installation deadline. The source-defined
capture inputs are 18 seconds, 96-byte snaplen, an 8-second readiness wait and
40-second overall context. The original source/destination/port/device and
counter results are in the structured extract; these are TEST ONLY addresses.

**Fixture activity can consume enough CPU, process-start or kernel time to
make a real 100 ms deadline fail.** Docker exec/copy, building the injector,
creating its capture sidecar and concurrent nft reads are possible load
sources. No CPU/memory limit is set by those shared commands. The original
capture does not retain effective cgroup limits, contemporaneous host load,
exact injection monotonic brackets, or the nft error/exit/signal. Thus neither
resource exhaustion nor injection causality is established. Even an nft error
unrelated to timing can be observed after a scheduling overrun. The complete
original generation-13 model and rendered transaction were not retained and
cannot be recreated exactly from a later live stack.

Agent-core 29 can use the original eight `install_ok=false` / over-budget
records to investigate the guarded installation boundary. Any extra command
error/exit instrumentation belongs in `node/enforce/executor.go` and its
producer timing path, outside this lane. No such product edit or assertion
relaxation was made. The admitted-sibling row stays failed.
