# lab-AT12

Mechanism: cross-member, management plane and DNS exceptions

Status: pass

Run: 5e9e4041-d942-4562-9e99-489bf735636a; started UTC: 2026-10-31T18:42:10Z; source dirty: false

Spec: §19 AT12, ISO01, ISO04, DNS03

UTC: 2026-10-31T18:42:14Z

Build: d9b770e06a3510880236eed57fe229465947951e

Image digests:

- fvpn-w4-isolation-lab:lab sha256:2c2b1647185ba2d99aa6fbd5a12f89939621fba5a3e11cf1709a8bb63223b7c8

Policy revision: generation 12

Node incarnation: mechanism-lab

Expected: Only TCP and UDP 53 to the designated resolver succeed. Peer, SSH, agent API, arbitrary overlay and a spoofed source are denied.

Actual: Two members and two devices of member A: peer, management, overlay, rogue and spoofed source denied. TCP and UDP 53 to 198.51.100.53 allowed. Host 127.0.0.1:2222 stayed up. Mechanism lab only.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- capture_direction: a2 inout equals out (a2-to-a1 SYN); a1/a2/b1/lan/inet ingress empty; three peer ingress positive controls observed; see lab-AT12-captures.txt
- resolver: 198.51.100.53 tcp/53 udp/53 allow; tcp/8080 deny
- spoof_source: 100.64.7.7 denied
- members: fv-a1 fv-a2 fv-b1
- scope: mechanism lab; product status is AT12.json
