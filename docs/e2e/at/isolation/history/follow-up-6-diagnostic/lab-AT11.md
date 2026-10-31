# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: 82eac082-27a1-4526-b1a4-ae05246824b2; started UTC: 2026-10-31T04:18:32Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-31T04:18:37Z

Build: 5ddd1126ff1ea72799384b9543f80419cfad4ebf

Image digests:

- fvpn-w4-isolation-lab:lab sha256:8c867df4e6a12cbad7a0e682745225f6d14fb7d3896384b40af6f9fc3f2b02df

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
