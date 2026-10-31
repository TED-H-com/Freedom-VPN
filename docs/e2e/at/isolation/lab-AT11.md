# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Run: edd986c4-16d2-4aab-b92f-baa19f8bd829; started UTC: 2026-10-31T21:46:28Z; source dirty: false

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-31T21:46:30Z

Build: 2304f77a733978405a35eea3b8d0a075027ce88e

Image digests:

- fvpn-w4-isolation-lab:lab sha256:dc7e12368cae0512b0f84a1d614de232a3713bcb37409e7db8e4fe93bca9de74

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
