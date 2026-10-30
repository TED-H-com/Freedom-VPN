# Follow-up 9: member traffic, hop by hop

The first stopped hop was **native member exit selection**, owned and fixed by
**w3-devstack-e2e**. The pinned NetBird client advertised the product-created
`0.0.0.0/0` network as `Not Selected`. The diagnostic request used the member's
lab bridge (`eth0`) and timed out. `netbird networks select all` still left the
exit unselected; explicitly selecting its actual `fvpn-exit-...` identifier
installed the tunnel route. No product forwarding rule was changed.

All addresses below belong to this isolated TEST ONLY lab. Captures print only
headers for the synthetic origin TCP/443 flow; no payloads or DNS names are kept.
The final complete run is `22c14cd9-39fc-444b-a7c5-c04cf3ffbff6` at `43176b8860bf90970e96b100c1ba4e957f09363d`.
[Full member evidence](runs/2026-10-30/member-traffic.json) includes every forward
rule, its counter reference, all named counter values, allowlist timeouts and
before/after interface statistics.

## Member

```text
192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache

Available Networks:

  - ID: fvpn-exit-19d4da50367ad609033ca17c
    Network: 0.0.0.0/0
    Status: Not Selected

Available Networks:

  - ID: fvpn-exit-19d4da50367ad609033ca17c
    Network: 0.0.0.0/0
    Status: Selected

192.0.3.10 dev fvnb0 table netbird src 100.104.75.121 uid 0
    cache
3: fvnb0    inet 100.104.75.121/16 brd 100.104.255.255 scope global fvnb0\       valid_lft forever preferred_lft forever
```

## Captured path and origin

| Hop | Captured TCP packets |
|---|---:|
| Member `fvnb0` | 16 |
| Node `fvmb0` | 16 |
| Node `fvup0` | 16 |

```text
conntrack v1.4.8 (conntrack-tools): 1 flow entries have been shown.
ipv4     2 tcp      6 2 CLOSE src=100.104.75.121 dst=192.0.3.10 sport=56150 dport=443 src=192.0.3.10 dst=192.0.3.20 sport=443 dport=56150 [ASSURED] mark=0 use=1
origin: {"probe_id":"ef9c25a6-39c0-40c6-a863-08b226040d2f","requests":1,"source_address":"192.0.3.20"}
```

The original tuple uses the member tunnel source; the reply tuple targets the
node's SNAT address. The authenticated origin slot records that same address.
The SNAT rule has **no counter expression** in the product renderer. Its raw
rule is retained; uplink capture, conntrack and origin arrival prove translation
without adding a rule or changing enforcement.

## Forward-chain counter deltas

Named counters shared by two rules remain shared; they are not reported as
independent per-direction measurements. Marker rules have no counter. The
`admission_bytes_after` reading is taken when wget returns; the later nft snapshot
follows the bounded capture and can include the final TCP close packets.

| Counter | Packets before → after | Bytes before → after |
|---|---:|---:|
| `ctbadns` | 0 → 0 | 0 → 0 |
| `ctbase` | 0 → 0 | 0 → 0 |
| `ctdeny` | 0 → 0 | 0 → 0 |
| `cthost` | 0 → 0 | 0 → 0 |
| `ctin` | 0 → 0 | 0 → 0 |
| `ctinv` | 0 → 0 | 0 → 0 |
| `ctnodn` | 0 → 0 | 0 → 0 |
| `ctunkd` | 0 → 0 | 0 → 0 |
| `ctunku` | 0 → 0 | 0 → 0 |
| `ctv6` | 0 → 0 | 0 → 0 |
| `d_01a18bc4cfe87c4c89c889b37cfec3f7` | 0 → 8 | 0 → 1518 |
| `nd_01a18bc4cfe87c4c89c889b37cfec3f7` | 0 → 0 | 0 → 0 |
| `nu_01a18bc4cfe87c4c89c889b37cfec3f7` | 0 → 0 | 0 → 0 |
| `u_01a18bc4cfe87c4c89c889b37cfec3f7` | 0 → 8 | 0 → 1113 |

Timed admission set after the request:

```json
{
  "family": "inet",
  "name": "allowed_v4",
  "table": "fvpn",
  "type": "ipv4_addr",
  "handle": 4,
  "flags": [
    "timeout"
  ],
  "elem": [
    {
      "elem": {
        "val": "100.104.75.121",
        "timeout": 3578,
        "expires": 3573
      }
    }
  ]
}
```

Device admission bytes: **0 → 2631**. HTTPS 200 plus counted admission: `true`. First stopped hop after the fix: **none after explicit NetBird exit selection**.
