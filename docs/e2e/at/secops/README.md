# Wave 4 security and operations acceptance

Follow-up 5 measured main `446b6b0` (#162), using only `w4-secops` / port base
`30000`. Run `86e6106a-1be1-4003-9b0f-35a30e9cda12` took place on a shared dev
host, **not candidate hardware**. [Dependencies](dependencies.json) list the
merged roadmap entries and the absent devstack follow-up 10.

```sh
make -f mk/at-secops.mk at-secops-followup5
```

The target refuses an occupied namespace and tears down its own objects. It
exits 2 while full acceptance has remaining gates; the live Go probe and all
25 supporting suites passed. The deadline worker guard is now started by the
target; this recorded run invoked the same guard alongside it before the faults.

| Case | Full status | Measured subset | Remaining gates |
|---|---|---|---|
| [AT28](AT28.json) | blocked | Usage 8 MiB and all other class cap triplets pass; duplicate retries leave one row; 20/20 login | Complete resource/log matrix; SECOPS-16 exhaustion recheck |
| [AT29](AT29.json) | blocked | Selected counted flow; paid agent-crash cutoff; grant kernel deadline with agent/watchdog stopped; valid-policy recovery | Finite grace input, short node-lease input, established TCP/UDP, guest clock rollback |
| [AT30](AT30.json) | blocked | Original physical PITR/synthetic replay retained, no new restore | Real node archive and independent provider/peer reconciliation |
| [AT31](AT31.json) | blocked | 20 readers and 2 counted devices, fixed five-minute window; real ledger/policy/usage observations | 60-device gateway budget, two nodes, forced relay/UDP, usable node resource telemetry, candidate hardware |
| [AT32](AT32.json) | blocked | Prior DOC01–14 fixes retained; DOC15 requests historical attribution of obsolete blockers | Three physical origins and native platform/hardware operator rows |

[Counted-flow proof](counted-flow.json) follows the required order: select the
single API-created `fvpn-exit-...`, route through `fvnb0`, verify HTTPS 200 and
an authenticated origin receipt from node SNAT, then check the exact device's
admission bytes. The first probe added **1,066 upload / 1,513 download bytes**.
Extra peers use the controlled IP in the origin certificate's SAN because they
do not inherit Compose hostname aliases. TLS verification stays enabled; no
route, group, policy or projected row is manufactured.

[Capacity](capacity.json) records 6,000 successful portal reads, zero wrong
member responses, p50/p95/p99 **13.442 / 23.074 / 368.457 ms**, and 300 successful
HTTPS requests from each native device. Admission deltas were
**310,512 / 456,974** and **307,780 / 453,957** upload/download bytes, including
protocol overhead and diagnostic policy probes. Policy ACKs took **6.013 s**
and **1.007 s**; latest sampled usage was API-visible within an observed
**4.371 s upper bound**. The node ledger had ACK62/max-seen62, 62 receipts and
123 declared/actual samples. The first member's total also includes the
separately labeled 300-byte synthetic node-B event. These numbers do not certify
throughput or the unrun full profile in [load-profile.json](load-profile.json).

There are 100 resource snapshots. Central-service CPU, memory and block IO are
retained; all node Docker samples returned 0B/0B memory and zero CPU/IO, which
is **unavailable telemetry**, not a zero-resource node. See [dev host](dev-host.json).

[Deadlines](deadlines.json) retain 66 timestamped HTTPS attempts, signed bounds,
monotonic heartbeat values, kernel timeout sets and recovery ACKs. Paid service
ended at 09:41:05Z; the last successful post-fault attempt started at 09:40:53.171Z
before watchdog closure. Grant service ended at 09:44:33Z; the last success
started at 09:44:31.763Z with the watchdog stopped. No attempt starting after
either end succeeded. The [worker guard](fault-injection.json) prevented central
reconciliation from being the cause of withdrawal. Original paid/grant ends
remained unchanged; only new finite grants and applied policy restored traffic.

[Post-revoke](post-revoke.json) records generation **22**, forwarding enabled,
empty block reasons, and a counted second-member flow after the first device's
202 revoke. The revoked device no longer advertised the exit. This resolves
the measured F3-02 coherence half after #153/#159/#162.

[Limiter](limiter.json) retires SECOPS-15 on measured cap−1/cap/cap+1 results.
[Attempt history](followup5-attempts.json) preserves the first complete run and
its extra-peer hostname failure. [Requests](requests.md), [scope review](scope-review.json),
[secret scan](secret-scan.json), [validation](validation.json) and
[operator procedures](operator-runs.md) retain qualifications and exact owners.
All live mutations use product APIs or exported lab/process hooks (D-050);
projected tables are read-only. No application, lab or migration source changed.
