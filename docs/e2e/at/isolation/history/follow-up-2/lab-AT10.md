# lab-AT10

Mechanism: host and upstream isolation during apply and interface change

Status: pass

Run: 422cdd6d-090b-4bb7-b92b-d4883259ac87; started UTC: 2026-10-30T17:28:15Z; source dirty: false

Spec: §19 AT10, ISO01, ISO02

UTC: 2026-10-30T17:28:21Z

Build: 11698fc4d7205a5c902592660ccb346d7f4db750

Image digests:

- fvpn-w4-isolation-lab:lab sha256:fbab877be24698a159f2db45530aa0943ac229100ba8d311ec52f0b9148bad2c

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
