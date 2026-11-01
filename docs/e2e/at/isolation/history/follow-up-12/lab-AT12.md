# lab-AT12

Mechanism: cross-member, management plane and DNS exceptions

Status: pass

Run: 1c25ee44-1872-4cce-a9f5-d52491d87095; started UTC: 2026-11-01T09:39:27Z; source dirty: false

Spec: §19 AT12, ISO01, ISO04, DNS03

UTC: 2026-11-01T09:39:31Z

Build: 99e58bff0cf7276e36cb04f35605835208c8638e

Image digests:

- fvpn-w4-isolation-lab:lab sha256:3d3bfa3b2e36ea184825464c02464b8fe4daae00ff6306c78705b11b4406e6ed

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
