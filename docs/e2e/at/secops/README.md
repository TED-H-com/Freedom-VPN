# Wave 4 security and operations acceptance

[Follow-up 6 summary](summary.md) records main `00e1d0c`, devstack #170 and
control #169. Two fresh `w4-secops` / 30000 instances each admitted **20 users
and 60 devices**, with 60 simultaneous current bindings. Both 600-read windows
returned HTTP 200 with no foreign devices; read p95 was **16.413 / 47.604 ms**.
These are admission and control diagnostics on a shared dev host.

| Case | Full status | Current result / remaining gate |
|---|---|---|
| [AT28](AT28.json) | blocked | Anonymous saturation leaves admin/lease 200; usage boundary 200/200/413, one duplicate-safe ledger row. SECOPS-16 stays open for positive protected transient key counts; full resource matrix and later trusted-proxy row remain. |
| [AT29](AT29.json) | blocked | Historical paid/grant cutoff subsets retained. Finite-grace prepare option, D-065 short fleet lease setter, established TCP/UDP and guest rollback remain. |
| [AT30](AT30.json) | blocked | Unchanged; backups did not change since #165. Real retained-node replay and independent provider/peer reconciliation remain. |
| [AT31](AT31.json) | blocked | SECOPS-10 admission resolved. Counted flow per peer, node process/cgroup telemetry, concurrent mixed/relay load and candidate hardware remain. |
| [AT32](AT32.json) | blocked | Historical scope review retained; DOC15 fixed by #163. Physical origins and platforms need operator runs. |

[SECOPS-16 records](secret-scan.json) cover 135 retained streams per phase and
protected transient streams. Retained streams have zero known-canary and
device-flow-key matches. The sequential profile has 60 exact pre-delete scans,
each with one device-flow key line. The helper reports 60 protected pre-delete
streams; an independent watcher observed a key in each, while the helper does
not return per-stream final key totals. No values or matched lines are public.
The earlier gateway finding is not retired under the requested all-stream-zero
criterion; current gateway diagnostics alone contain zero key/exhaustion lines.

Use [RUN.md](RUN.md), [requests.md](requests.md), [dependencies.json](dependencies.json)
and [validation.json](validation.json). The run target exits 2 for full-case
gates while both live probes pass. Both instances were torn down. Prior
counted-traffic, deadline, revocation and PITR measurements remain attributed
under [history/follow-up-5](history/README.md); they are not new 60-peer traffic
measurements. [Gateway inventory](gateway-inventory.json) qualifies the initial
unavailable worker count and preserves socket/FD samples without addresses.
