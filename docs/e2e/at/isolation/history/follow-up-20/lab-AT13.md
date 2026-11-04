# lab-AT13

Mechanism: revocation, agent SIGKILL and baseline

Status: pass

Run: 70beb28e-4766-4004-932b-510fd01f986a; started UTC: 2026-11-03T22:10:08Z; source dirty: false

Spec: §19 AT13, ISO02, ENT02, SEC01

UTC: 2026-11-03T22:10:14Z

Build: 458b7a89f07c22dd4e66f839195b6549db4b883d

Image digests:

- fvpn-w4-isolation-lab:lab sha256:2402d3a556b748a875bd1fa2660bb7e9fad666fb4a2e7c7e65b9d8f31ce9cdcf

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

## Follow-up 19 — D-147 and assembled limits (2026-11-03)

[Dated D-147 evidence](d147-related-frag-needed-2026-11-03.md) records four independent source-set positives, exact capture/counter/class reconciliation and all three policy negatives. It has its own [lab record](lab-AT13-d147.json), separate from this original AT13 record. Literal `ctdeny` is not exercised for exclusive resolver/host rows; their observed counters are `ctbadns` / `cthost` (F19-01).

The single assembled run `70beb28e-4766-4004-932b-510fd01f986a` is blocked at serving setup after a shared-publisher capacity PATCH 412. No real-node expiry, revoke, restart or full fault matrix was exercised; no prior pass is promoted. Its initial shared-host window is confounded under D-133 and was not retried. [Summary](summary.md), [requests](requests.md). Earlier records remain in [follow-up 18](history/follow-up-18/archive.json); focused lab records and failed attempts remain under `history/follow-up-19-*`.
