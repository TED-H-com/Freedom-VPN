# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: bd73d7a4-03f1-4b17-97c2-9c25a39ac0db; started UTC: 2026-11-04T15:48:54Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-11-04T15:48:57Z

Build: 4c495c3b1516022e1de8db2f864432a8a8d171c5

Image digests:

- fvpn-w4-isolation-lab:lab sha256:b53c2651ca9050eff8ba6f9ba711f865ce7739b4970d80379700ab7ce24e2dbc

Policy revision: generation 11

Node incarnation: mechanism-lab

Expected: IPv6 mode block drops member IPv6, including ULA, link-local, loopback and IPv4-mapped. Metadata stays unreachable. There is no route mode in v1.

Actual: mode block, ctv6 +3, special-address probes denied, target captures empty. Route mode was not claimed (§8.3).

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- ipv6_mode: block
- ctv6_delta: 3
- route_mode: not implemented in v1
- scope: mechanism lab; product status is AT11.json
