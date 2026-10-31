# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: c0106e5d-d8e8-400b-abe0-181d6428fc55; started UTC: 2026-10-31T01:57:42Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-31T01:57:45Z

Build: 7fa0443c532b1c9a395cd93620152fb6902584ae

Image digests:

- fvpn-w4-isolation-lab:lab sha256:7c4d3eabd27e2cb8d996dd644e48526fb3aeace6e2050c03783974cb4abb2475

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
