# Follow-up 4 forwarding observation

Run `d89eda32-84ef-4582-b669-053d5d9abadd`; UTC timestamps, source/test/probe hashes, image digests, policy
revisions and both incarnations are in each linked JSON. Only controlled-origin
TCP/443 aggregate packet counts are retained; no payload or credentials.

The normal serving path now counts member traffic. The native client waits for
the advertised route and explicitly selects the `fvpn-exit-...` network matching
the intended node's real peer, preserving `skip_auto_apply=true`. A positive
requires the exact CA-verified origin response, authenticated origin receipt
from that node's SNAT address, and increases in that device's upload and download
counters. The shared forwarding driver's selection requirement is confirmed by
[RUN.md](../../RUN.md#native-exit-selection-and-traffic-trace); this harness also
uses bounded advertisement polling, covering the shared driver's documented
immediate-discovery timing limit.

| Case | Counted positive transfers | Other observed transfers |
|---|---:|---|
| [AT01](AT01.json) | 2 | Before/after Linux pause/unpause |
| [AT02](AT02.json) | 5 | Race loser and revoked peer: no origin receipt |
| [AT03](AT03.json) | 1 | Device admitted through actual paid reconciliation |
| [AT04](AT04.json) | 6 | Six new-connection denials after refund/expiry/suspension; persistent timing in summary |
| [AT08](AT08.json) | 2 | Four unauthorized/disabled denials, including after restart |
| [AT09](AT09.json) | 3 | One failed second-exit recovery transfer, preserved |

AT02's transaction outcomes are one winner/one quota error and 3→2→3 slots.
After DELETE the former loser and a sibling carry counted traffic, revoked
source conntrack is empty, accepted health agrees, and the agent PID is
unchanged. No restart/state clear or D-050 repair is used. This passes §19 AT02's
stated quota/ownership/withdrawal basis. The strict post-revoke node-ingress drop
measurement is the [shared devstack limitation](../../runs/2026-10-30/followup10-run3/device-revoke.md),
not a claim made here. The updater-only probe is not used as member traffic.

## Open restart condition: F4-01

Owner: **w2-agent-core**, with **w3-wire-nodes** lease issuance. After a counted
node2 flow, the second attempt stops the node, resolver and namespace holder.
The offline transfer has zero origin receipts. On restart, the same incarnation
gets a fresh generation-23 lease at `17:37:24.662928Z` with zero member grants;
its expiry is the following day. The preceding generation-23 lease had seven.
The node fails the 90-second fresh-serving gate and reports `lease_expired`.

At `17:38:54Z`, the selected member route uses `fvnb0`. Five TCP attempts reach
node2 `fvmb0`; its generation is 0 before and after, member accept rules are 0,
`ctbase` increases from 0 to 300 bytes, device counters do not increase, and the
origin receives nothing. **The first stopping point is the node forward-chain
baseline**, not the member route or the origin. The primary node subsequently
still carries a counted transfer.

AT20's independent DNS revision then changes the desired generation to 24,
receives a seven-member lease at `17:39:36.696084Z`, and observes forwarding
true. Later read-only evidence must not pass the failed AT09 recovery. The
source points to `needRenew` retaining a fresh same-generation empty grant
lease until its renewal lead, with `Decide` labelling the empty admitted model
`lease_expired`; `RenewTx` gates grants on readiness. This is a supported
inference, not a claimed internal branch trace. Exact reproduction and requested
bounded refresh behavior are in [requests.md](requests.md#f4-01--empty-member-lease-after-second-exit-restart).

Attempt 1 used only an agent-container restart, leaving namespace holders alive,
and its immediate recovery sample raced fresh readiness. That distinct earlier
result remains [archived](history/follow-up-4/attempt-1/AT09.json). No fault,
failed assertion or silent later pass is substituted for the second attempt.
