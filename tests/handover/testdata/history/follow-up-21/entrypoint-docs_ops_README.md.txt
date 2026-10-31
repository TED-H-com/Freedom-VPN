# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 20](../handover/follow-up-20.md) (2026-10-31 EDT):** Acceptance through #344. Updates F10-02, F10-01 shared side and the D-087 central seam are closed; ISO-F6-03 remains open. Four acceptance sessions have no merged results at this cutoff.
Previous cut: [follow-up 19](../handover/follow-up-19.md), through #337.

## Follow-up 20 acceptance snapshot (2026-10-31 EDT)

Fixed main cutoff `96ec0630`, through **#344**. See the
[full handover](../handover/follow-up-20.md) and
[decision / findings register](../handover/README.md#follow-up-20-decisions-register-2026-10-31-edt).
This table consumes merged evidence; handover ran no new live acceptance.

| Delivery | Measured scope at the cutoff |
|---|---|
| Usage 6 (#338) | AT14–AT20 pass, run 71b1a160. Production planBaseline gap sequence 1 / zero records / 1 µs reaches ACK 1 before traffic; event 2 is 1,066 + 1,513 = 2,579 bytes. D-130 newly minted-event measurement closed; follow-up 5 archived (59 files). |
| Handover 19 (#339) | Frozen #337 cut: 16 subtests / 13 roadmap rows / 37 immutable inputs; D-129–D-132 and migration 00193 rollout / Down guidance preserved. |
| Fleet-update 13 (#340) | D-132 owned D062 fixture copies, empty-result failure and mutable-path guard; measured D-110 / D-127 central acceptance recorded. No product change. |
| Installer 7 (#341) | Updates F10-02 FIXED; generated HTTPS origin / server CA / mTLS reporting, exact control-origin allowlist, TEST ONLY loopback opt-in; privileged 13/13. Distinct artifact host remains an owner decision for updates 11. |
| Updater 18 (#342) | Missing sink fails explicit flush; result_url_missing keeps a sealed receipt pending, repaired config sends it once. Root executable / whole-root integrity fixtures; race/load 40/40. F18-01 devstack confirmation pending. |
| Devstack 20 (#343) | D-087 seam 7/0/0 twice, 3,980 / 4,112 ms; F10-01 shared side FIXED, switching 11/0/0. Counter diagnostics 247 samples; DNS 36 complete traces. Broader 15/0/1 retains instrumented MTU-1280 failure 10,010 / 64,064 bytes; F8-02 still unattributed. |
| Agent-core 28 (#344) | Verified same-boot adoption and same-lease repair; new baseline/heartbeat/watchdog evidence contract. ISO-F6-03 writer unattributed. D-133 A/B/A/B passes four live and four full agent suites; initial 152.125 ms failure attributed to host contention. Two measured limits stay open for netlab 21, not defects of record. README restart request closed. |

At #344, updates 11 (regular AT23–AT27 / artifact origin), isolation 10
(ISO-F6-03 / assembled AT10), members 11 (F10-02 / F10-03 / complete AT09 under
D-129), and dns-publish 9 (F10-03) are pending, with no results on main.
F18-01 belongs to devstack 21; F8-02 and D-133 limits to netlab 21. Native MTU
1280 under D-123, AT11 routes, AT13 continuous faults including physical arm64,
AT29 / AT31 resources, OS reboot, power loss and real platform clients stay open.
Dependabot #22 is an open medium minimatch dependency-hygiene item for portal,
pending reachability assessment. No acceptance tolerance or product value changed.

Historical follow-up 17 reconciliation: [follow-up 17](#follow-up-17-acceptance-snapshot-2026-10-31-edt),
main `7e044c71` through #313. Earlier tables retain the verdicts at their own cutoffs.

Operational runbooks, deployment guides, parameter calibration references, and disaster recovery checklists are maintained under [`docs/handover/`](../handover/README.md) alongside domain-specific operational references:

## Hand-Over & Production Runbooks (`docs/handover/`)
