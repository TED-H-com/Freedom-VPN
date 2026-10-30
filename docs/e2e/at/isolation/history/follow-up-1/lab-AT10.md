# lab-AT10

Mechanism: host and upstream isolation during apply and interface change

Status: pass

Spec: §19 AT10, ISO01, ISO02

UTC: 2026-10-30T00:05:02Z

Build: 4737a6ec3533f230dddb61a5b5e2a18b7dddd580

Image digests:

- fvpn-w4-isolation-lab:lab sha256:3d04195846457be0a66281213c93c446308563faa8c0372a9a08acf039a5ac64

Policy revision: generation 3000-3020

Node incarnation: mechanism-lab

Expected: Protected IPv4 destinations stay denied while the ruleset is regenerated and a lab veth appears. The controlled Internet endpoint stays reachable. The probe holds at least ten attempts per second and records no allow.

Actual: 20 regenerations, interface flap, rate 86.6 attempts/s, protected destinations denied, 203.0.113.10:80 allowed, captures empty. This is the mechanism lab, not the assembled stack.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- attempts: 273
- seconds: 3.151
- rate_per_s: 86.64
- regenerations: 20
- flap: veth fv-px 198.18.0.0/30 inside the lab container
- internet: 203.0.113.10:80 allowed
- scope: mechanism lab; product status is AT10.json
