# lab-AT13

Mechanism: revocation, agent SIGKILL and baseline

Status: pass

Run: 041e9e48-79a6-45dd-a19d-8b5a63290655; started UTC: 2026-10-31T06:15:42Z; source dirty: false

Spec: §19 AT13, ISO02, ENT02, SEC01

UTC: 2026-10-31T06:15:47Z

Build: 15ce5f60a1c39d194ef9544a29a02fe40575ab84

Image digests:

- fvpn-w4-isolation-lab:lab sha256:20ab0acdd306acdf25186f86060bbba4a29c8d4c2a79ae8ed7d20dfa6aadce7e

Policy revision: generation 4000-4100

Node incarnation: mechanism-lab

Expected: An established flow dies on revoke, conntrack for that source is removed, SIGKILL leaves the ruleset, and baseline keeps member forwarding off.

Actual: Revoke stopped the live flow and cleared its conntrack. SIGKILL left generation 4100. Baseline returned generation 0 and denied both members while 127.0.0.1:2222 stayed up. NetBird restart, node reboot and the updater switch are not in this container.

Commands:

- `FVPN_LAB_PREFIX=fvpn-w4-isolation-lab make -f mk/at-isolation.mk at-isolation-lab-up`
- `FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-w4-isolation-lab FVPN_AT_EVIDENCE_DIR=docs/e2e/at/isolation go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at`

- conntrack: 100.64.1.1 removed after revoke
- sigkill: generation 4100 remained
- baseline: generation 0, member forwarding denied
- not_exercised: NetBird restart, node reboot, updater switch
- scope: mechanism lab; product status is AT13.json
