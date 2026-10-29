# lab-AT11

Mechanism: IPv6 block mode, metadata and special addresses

Status: pass

Spec: §19 AT11, ISO01, ISO03, §8.3

UTC: 2026-10-29T23:40:50Z

Build: c1ac4393f6c206783f06034d73f2915d675bd917

Image digests:

- fvpn-w4-at-isolation:lab sha256:fb7de817d87b65403435999fd8676cf58e6acb3769a2023d6da0b2fdd7120d46

Policy revision: generation 11

Node incarnation: mechanism-lab

Expected: IPv6 mode block drops member IPv6, including ULA, link-local, loopback and IPv4-mapped. Metadata stays unreachable. There is no route mode in v1.

Actual: mode block, ctv6 +3, special-address probes denied, target captures empty. Route mode was not claimed (§8.3).

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-at-isolation make -f mk/netlab.mk netlab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-at-isolation FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- ipv6_mode: block
- ctv6_delta: 3
- route_mode: not implemented in v1
- scope: mechanism lab; product status is AT11.json
