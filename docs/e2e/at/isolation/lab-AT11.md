# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 70beb28e-4766-4004-932b-510fd01f986a; started UTC: 2026-11-03T22:10:00Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-11-03T22:10:03Z

Build: 458b7a89f07c22dd4e66f839195b6549db4b883d

Image digests:

- fvpn-w4-isolation-lab:lab sha256:2402d3a556b748a875bd1fa2660bb7e9fad666fb4a2e7c7e65b9d8f31ce9cdcf

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
