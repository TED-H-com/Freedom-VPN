# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 9c342b4f-de38-47c9-807c-3be32fbe677c; started UTC: 2026-11-03T22:04:15Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-11-03T22:04:18Z

Build: 486d75d189e459ac619f7f228bdf802eec239f52

Image digests:

- fvpn-w4-isolation-lab:lab sha256:830c7fc8f9ac635063bc720a850c9443ffc08e1d6b667c635d17e7c87657c7a9

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
