# lab-AT10

Mechanism: host and upstream isolation during apply and interface change

Status: pass

Run: 041e9e48-79a6-45dd-a19d-8b5a63290655; started UTC: 2026-10-31T06:15:27Z; source dirty: false

Spec: §19 AT10, ISO01, ISO02

UTC: 2026-10-31T06:15:34Z

Build: 15ce5f60a1c39d194ef9544a29a02fe40575ab84

Image digests:

- fvpn-w4-isolation-lab:lab sha256:20ab0acdd306acdf25186f86060bbba4a29c8d4c2a79ae8ed7d20dfa6aadce7e

Policy revision: generation 3000-3020

Node incarnation: mechanism-lab

Expected: Protected IPv4 destinations stay denied while the ruleset is regenerated and a lab veth appears. The controlled Internet endpoint stays reachable. The probe holds at least ten attempts per second and records no allow.

Actual: 20 regenerations, interface flap, rate 86.5 attempts/s, protected destinations denied, 203.0.113.10:80 allowed, captures empty. This is the mechanism lab, not the assembled stack.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- tc_readback: [{"interface":"fv-fh","attachment":"qdisc","objects":[{"kind":"htb","handle":"1:","root":true},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"fq_codel","handle":"2f:","parent":"1:2f"}]},{"interface":"fv-fh","attachment":"root","objects":[]},{"interface":"fv-fh","attachment":"ingress","objects":[]},{"interface":"fv-fh","attachment":"egress","objects":[]},{"interface":"fv-fh","attachment":"1:1","objects":[]},{"interface":"fv-fh","attachment":"1:2f","objects":[]},{"interface":"fv-fh","attachment":"1:30","objects":[]},{"interface":"fv-fh","attachment":"1:fffe","objects":[]},{"interface":"fv-fh","attachment":"2f:","objects":[]},{"interface":"fv-fh","attachment":"30:","objects":[]},{"interface":"fv-ih","attachment":"qdisc","objects":[{"kind":"htb","handle":"1:","root":true},{"kind":"fq_codel","handle":"30:","parent":"1:30"},{"kind":"fq_codel","handle":"2f:","parent":"1:2f"}]},{"interface":"fv-ih","attachment":"root","objects":[]},{"interface":"fv-ih","attachment":"ingress","objects":[]},{"interface":"fv-ih","attachment":"egress","objects":[]},{"interface":"fv-ih","attachment":"1:1","objects":[]},{"interface":"fv-ih","attachment":"1:2f","objects":[]},{"interface":"fv-ih","attachment":"1:30","objects":[]},{"interface":"fv-ih","attachment":"1:fffe","objects":[]},{"interface":"fv-ih","attachment":"2f:","objects":[]},{"interface":"fv-ih","attachment":"30:","objects":[]}]
- attempts: 336
- seconds: 3.884
- rate_per_s: 86.51
- regenerations: 20
- flap: veth fv-px 198.18.0.0/30 inside the lab container
- internet: 203.0.113.10:80 allowed
- scope: mechanism lab; product status is AT10.json
