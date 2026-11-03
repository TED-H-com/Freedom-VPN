# lab-AT10

Mechanism: host and upstream isolation during apply and interface change

Status: fail

Run: c32f1cab-f62e-4a73-ac4d-5452f71e052d; started UTC: 2026-11-03T22:04:10Z; source dirty: false

Spec: §19 AT10, ISO01, ISO02

UTC: 2026-11-03T22:04:15Z

Build: 486d75d189e459ac619f7f228bdf802eec239f52

Image digests:

- fvpn-w4-isolation-lab:lab sha256:830c7fc8f9ac635063bc720a850c9443ffc08e1d6b667c635d17e7c87657c7a9

Policy revision: generation 3000-3020

Node incarnation: mechanism-lab

Expected: Protected IPv4 destinations stay denied while the ruleset is regenerated and a lab veth appears. The controlled Internet endpoint stays reachable. The probe holds at least ten attempts per second and records no allow.

Actual: allowed Internet flow failed during regeneration

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`
