# IPv6 handshake response after a successful first selection

2026-11-01. [Packet and counter joins](loopback-underlay.json),
[corrected initiation origins](initiation-origins.json),
[complete timeline](transport-timeline.md), and
[all stream sends](stream-intervals.csv).

This row has one underlay type-2 response with source `::1`, one observed member
IPv6 header-error increment, a successful first selection, and no missing node
ingress among 7,322 stream sends. The response did not precede the first
successful delivery:

| Unix seconds | Observation |
|---|---|
| 1793547451.716756 | First inner SYN leaves member, packet 455. |
| 1793547451.716846 | Member initiation `e85074a2`, sender index 1214008509. |
| 1793547451.718879 | Node initiation `25905953`, sender index 3178637190. |
| 1793547451.719596 | Member response `36383e6d`, receiver 3178637190, reaches node at .720112. |
| 1793547451.720285 | Member data `57303ac0`, receiver 3178637190 / counter 0; identical bytes reach node at .720363. |
| 1793547451.720375 | The matching inner SYN reaches node, packet 169: 3.585356–3.630897 ms after member egress. |
| 1793547451.920478 | The original member initiation `e85074a2` reaches node loopback as IPv6. |
| 1793547451.920671 | Node `eth2` packet 285 sends response `6081d9e3` from `::1`; member `eth1` packet 640 receives identical bytes at .920681. |
| 1793547467.316916 | Next real member initiation `3597bf0a`, 15.600070 s after its previous initiation; normal-source response reaches member at .317184. |
| 1793547474.934974229 | First sustained-stream socket write starts, after an independent serving gate. |

The encrypted receiver-index join and subsequent inner SYN support delivery
through the earlier node-originated handshake. No ciphertext is decrypted by
the observer. The bad-source response follows the successful SYN by about
200 ms. Its counter-error window is `after-first-transfer` →
`after-passive-tail`. The stream starts after the later handshake, so its lack
of loss cannot establish uninterrupted forwarding during that earlier idle
interval. The first-selection request itself completed before the anomaly.

The record does not establish a five-second retry, an exact kernel discard
callsite, or a common cause for historical F11-01, AT02 and F14-01. It does show
why response direction, concurrent handshakes and traffic timing must be joined
before assigning a delay. The member route tables, both endpoint/status
histories and independent latest-handshake readbacks remain in the full capture.
