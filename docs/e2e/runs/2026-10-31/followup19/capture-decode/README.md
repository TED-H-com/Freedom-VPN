# F8-02 retained capture decode (TEST ONLY)

All four raw files match the requested byte counts and SHA-256 pins. The
JSONL files retain every captured record, TCP sequence/ACK/SACK/window/options
and lengths, and outer UDP/IP DF/fragment identifiers/offsets. Payloads and
MACs are omitted; addresses are campaign-stable endpoint tokens. Interface
indices are the original Linux cooked v2 indices (packet type 4 outbound,
0 host inbound); current host interface names must not be substituted.
Original read-backs identify member 10=fvnb0, node 6=fvmb0 and 3=fvup0.

Each file ends on a complete record, with zero truncated headers. Snaplen 128
truncates packet data intentionally; captured IP/TCP lengths are retained.
GSO/GRO can produce captured IP lengths larger than interface MTU, so these
are capture observations, not proof of on-wire oversized packets.

```json
[
  {
    "file": "2454c833-ae93-48e0-96b0-00dee7cc3117-ipv4_1280-member.pcap",
    "bytes": 35536,
    "sha256": "a5b7059d092c1a7352769fa1be874d0820982cd610165aa141e20f773e59f9ec",
    "snaplen": 128,
    "linktype": 276,
    "records": 265,
    "snap_truncated_records": 183,
    "header_errors": 0,
    "unparsed_records": 0,
    "file_complete": true,
    "first": "1793442772.722932",
    "last": "1793442784.930122",
    "protocol_records": {
      "17": 153,
      "6": 60,
      "fragment": 52
    }
  },
  {
    "file": "2454c833-ae93-48e0-96b0-00dee7cc3117-ipv4_1280-node.pcap",
    "bytes": 54892,
    "sha256": "17f0652be93b2a94844a056798df221fcd0c11673f98287f8b84381cfc8cd953",
    "snaplen": 128,
    "linktype": 276,
    "records": 407,
    "snap_truncated_records": 305,
    "header_errors": 0,
    "unparsed_records": 0,
    "file_complete": true,
    "first": "1793442772.722943",
    "last": "1793442784.930114",
    "protocol_records": {
      "17": 168,
      "6": 186,
      "fragment": 52,
      "58": 1
    }
  },
  {
    "file": "daf790fc-81f7-49a2-8a65-f3ec8faec656-ipv4_1280-member.pcap",
    "bytes": 44388,
    "sha256": "2f66fa0e576f68e19d74796c2c30f44327e72c108660de63cf00ede08588efdc",
    "snaplen": 128,
    "linktype": 276,
    "records": 334,
    "snap_truncated_records": 234,
    "header_errors": 0,
    "unparsed_records": 0,
    "file_complete": true,
    "first": "1793443233.909154",
    "last": "1793443234.029177",
    "protocol_records": {
      "17": 165,
      "6": 117,
      "fragment": 52
    }
  },
  {
    "file": "daf790fc-81f7-49a2-8a65-f3ec8faec656-ipv4_1280-node.pcap",
    "bytes": 67414,
    "sha256": "ca9390d8bfa02febe5b2e9403b60d0f2263a0baacafe435b582eb27f23aaa71d",
    "snaplen": 128,
    "linktype": 276,
    "records": 509,
    "snap_truncated_records": 371,
    "header_errors": 0,
    "unparsed_records": 0,
    "file_complete": true,
    "first": "1793443234.027446",
    "last": "1793443234.365069",
    "protocol_records": {
      "17": 165,
      "6": 290,
      "fragment": 52,
      "58": 2
    }
  }
]
```

The failed flow is port 41478, member ISN 2418164341 / origin ISN 10709237.
The member ACK stalls at **10731373**, after `3 + 22*1006` stream bytes: OK
plus 22 complete ECHO lines, consistent with **22,022** validated payload
bytes. SACK reports later ranges including `[10736285,10739421)` and
`[10744333,10744451)`. The node captures repeated 1228-byte transmissions
beginning at 10731373 on fvup0 ingress and fvmb0 egress; the member capture
never receives them. Its advertised window is nonzero. This localizes the
observed missing range beyond the node's tunnel egress capture, and disproves
"all bytes arrived, client counted only one recv" for this pair. It does not
prove the precise outer fragment/native receive failure point.

The control is port 55090, ISNs 4196169297 / 59143023. Member and node retain
all 64,387 origin stream bytes (OK plus 64 ECHO lines), cumulative ACK
59207411 and FIN exchange. Both handshakes retain MSS 1240/1460 unchanged.
The older failure also contains a separate prior connection (port 44904);
that traffic must not be joined to port 41478. No zero-window explanation
or MSS-clamp change is established.

The failed capture's tcpdump start/stop/kernel-drop statistics were not retained
in follow-up 18. File completeness does not fill that gap. Its first/last packet
timestamps above are observed capture bounds, not invented process start/stop
times. The control's original stats are 334/509 packets and zero kernel drops,
matching decoded totals. The original 11,011-byte members row has **no matching
pcap in this worktree**; the available 22,022-byte row is a different run. A
fresh same-order native reconnect campaign is recorded separately with origin
headers and endpoint samples. No historical verdict is replaced.
