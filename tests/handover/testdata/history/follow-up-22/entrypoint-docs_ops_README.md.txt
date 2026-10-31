# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 21](../handover/follow-up-21.md) (2026-10-31 EDT):** Through #351: positive-cache limits, retained verified install archives and Dependabot #22 status are documented. Regular AT23–AT27 and assembled D-087 AT10 pass; historical failures remain frozen.
Previous cut: [follow-up 20](../handover/follow-up-20.md), through #344.

## Follow-up 21 acceptance snapshot (2026-10-31 EDT)

Fixed main cutoff `6d1e320d`, through **#351**. Handover ran no new live
acceptance. See [full evidence](../handover/follow-up-21.md) and the
[decision / findings register](../handover/README.md#follow-up-21-decisions-register-2026-10-31-edt).
No new decision; D-133 remains the latest. Devstack 21, netlab 21 and agent-core
29 were not on main at start; their results are excluded.

| Delivery | Measured scope at the cutoff |
|---|---|
| Dns-publish 9 (#345) | F9-HIT / F9-OTHER / F9-ROUND and inherent F9-INFLIGHT on pinned defaults; TTL is a maximum lifetime. Mandatory dns-positive-cache gate. Historical F10-03 failed / unattributed; blocked_response_ttl still owner's. |
| Updates 11 (#346) | daddad24: installer 27 / 27; regular AT23–AT27 5 / 5, physical rollback witnessed by AT27; generated reporting without adapter, 13 POSTs. Artifact-origin question closed by measurement, no rejection or exception needed. F18-01 remains negative; updates F11-01 to devstack 21. |
| Handover 20 (#347) | Frozen #344 cut: 18 subtests / 7 pinned rows / 38 immutable inputs, D-133 registration retained. |
| Members 11 (#348) | 7edb263f: F10-02 new row 186 / 186, F10-03 new row passes, historical failures unchanged; F10-04 closed. Full AT02 121 / 0 / 2 (F11-02 / F11-03); AT09 37 / 5 / 1 (F8-02 IPv6 recurrence); new F11-01 route gap. 2 pass / 6 blocked / 2 fail. |
| Portal 19 (#349) | Dependabot #22 cannot close from lockfile: full override breaks jsx-a11y; files byte-identical. Development only, no production-bundle path. portal-verify 74 / 74; portal-e2e 9 / 9; owner disposition pending. |
| Isolation 10 (#350) | bfcc3880: real D-087 AT10 PASS 9 / 0 / 0, 962.448 ms; AT12 PASS 10 / 0 / 0; AT13 BLOCKED 6 / 1 / 0, fresh D-104 passes but historical ISO-F6-03 writer unattributed; AT11 BLOCKED 2 / 1 / 0. Independent 968-guard distributions and both failed attempts retained. |
| Updater 19 (#351) | Installer-shaped config builds HTTPSink with exact policy and distinct server trust; no reporting-field divergence. Updates F11-02 offline paragraph fixed: verified archive retained, only absence downloads. D-132 audit zero docs/e2e reads; physical arm64/power loss/fleet/production/worst-case bounds still unmeasured. |

Open: ISO-F6-03 writer; F8-02 on both underlays; members F11-01 / F11-02 /
F11-03; updates F11-01; F18-01; Dependabot #22; two D-133 measured limits;
blocked_response_ttl; AT11 real platform routes; AT13 full fault matrix including
physical arm64; AT29 / AT31 resources; real clients and QUIC / PMTU tooling;
commercial feed approval. The [operator changes](../handover/operator-guide.md#operator-changes-at-351)
cover positive-cache semantics, retained verified archives and the dependency
alert only. The previous snapshot below remains historical at #344.
