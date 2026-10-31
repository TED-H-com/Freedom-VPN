# lab-AT10

Mechanism: host and upstream isolation during apply and interface change

Status: pass

Run: 17c14696-3eb1-469d-8aab-f947f6f425e4; started UTC: 2026-10-31T18:16:24Z; source dirty: false

Spec: §19 AT10, ISO01, ISO02

UTC: 2026-10-31T18:16:30Z

Build: 2da5acfbbe08f094fa46dc91802bf75012481f71

Image digests:

- fvpn-w4-isolation-lab:lab sha256:ecce4afe89e08a13c643ce6a93802aa8e433b757528b27c8a28c123540a499d4

Policy revision: generation 3000-3020

Node incarnation: mechanism-lab

Expected: Protected IPv4 destinations stay denied while the ruleset is regenerated and a lab veth appears. The controlled Internet endpoint stays reachable. The probe holds at least ten attempts per second and records no allow.

Actual: 20 regenerations, interface flap, rate 86.4 attempts/s, protected destinations denied, 203.0.113.10:80 allowed, captures empty. This is the mechanism lab, not the assembled stack.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- tc_readback: [{"interface":"fv-fh","attachment":"qdisc","objects":[{"kind":"fq_codel","handle":"2f:","parent":"1:2f"},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"htb","handle":"1:","root":true}]},{"interface":"fv-ih","attachment":"qdisc","objects":[{"kind":"fq_codel","handle":"2f:","parent":"1:2f"},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"htb","handle":"1:","root":true}]},{"interface":"fv-fh","attachment":"1:1","objects":[]},{"interface":"fv-fh","attachment":"1:2f","objects":[]},{"interface":"fv-fh","attachment":"1:30","objects":[]},{"interface":"fv-fh","attachment":"1:fffe","objects":[]},{"interface":"fv-fh","attachment":"2f:","objects":[]},{"interface":"fv-fh","attachment":"30:","objects":[]},{"interface":"fv-fh","attachment":"egress","objects":[]},{"interface":"fv-fh","attachment":"ingress","objects":[]},{"interface":"fv-fh","attachment":"root","objects":[]},{"interface":"fv-ih","attachment":"1:1","objects":[]},{"interface":"fv-ih","attachment":"1:2f","objects":[]},{"interface":"fv-ih","attachment":"1:30","objects":[]},{"interface":"fv-ih","attachment":"1:fffe","objects":[]},{"interface":"fv-ih","attachment":"2f:","objects":[]}]
- attempts: 364
- seconds: 4.214
- rate_per_s: 86.38
- regenerations: 20
- flap: veth fv-px 198.18.0.0/30 inside the lab container
- internet: 203.0.113.10:80 allowed
- scope: mechanism lab; product status is AT10.json
