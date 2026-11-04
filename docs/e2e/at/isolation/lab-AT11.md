# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: df9dbad7-049d-4eaa-ab98-5490906cc3e9; started UTC: 2026-11-04T14:06:43Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-11-04T14:06:46Z

Build: 56d814113913ddc6177d47d4a5160a65c93add25

Image digests:

- fvpn-w4-isolation-lab:lab sha256:9b53681fbed522d7b3a619f5991cfc66abb8fb4d9d82847bdb8fa39c73fc98b5

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
