# Wave 4 security and operations acceptance

Follow-up 4 uses the shared dev stack on `w4-secops` / port base `30000`.

```sh
make -f mk/at-secops.mk at-secops-followup4
```

The runner refuses an occupied namespace, creates fresh TEST ONLY state, runs
live probes, removes its runtime objects, and runs supporting suites with
`-p 1 -parallel 2`. Exit 2 preserves blocked acceptance; inspect individual
results rather than treating that exit as a pass. No application, contract,
migration or shared lab source is changed.

| Case | Evidence | Scope |
|---|---|---|
| AT28 | [record](AT28.json), [live](live.json), [limits](limiter.json) | Role/session/PKI/CSRF/ownership and secret scanning; new usage-size refusal is retained as a failure |
| AT29 | [record](AT29.json), [publication](publication.json) | Publication refusal/success and real readiness; deadline faults require a counted serving flow |
| AT30 | [record](AT30.json), [PITR](pitr.json) | Historical physical restore and synthetic replay; full node archive/external reconciliation remains blocked |
| AT31 | [record](AT31.json), [limits](limiter.json), [dependencies](dependencies.json) | Corrected D-060 classes, 20-user login, actual native-peer admission; no capacity certification without counted traffic |
| AT32 | [record](AT32.json), [review](scope-review.json) | DOC01–DOC14 resolved; platform/source/hardware rows remain needs operator run |

D-050: groups, domain rules, DNS/plan/fair-use policy revisions, activation and
finite grants use admin APIs. The DNS fixture blocks `blocked.example.invalid`
so resolver smoke probes remain within the controlled lab; an empty profile's
fallback `example.com` cannot pass this lab's NXDOMAIN-only public zone.
`e2e.PrepareServingBaseline` supplies signed binaries/support policy. Native
peers use the exported `StartNativePeer` / `JoinNativeMember` hooks and production
discovery. **No live SQL fixture writes remain.** Isolated PKI component identity
rows remain separate from serving evidence.

The publication negative uses a genuine ready baseline with capacity unset,
then records the failed operation step. The capacity command supplies TEST ONLY
200 Mbit/s parent limits and 100 Mbit/s member caps. A retrying operation may
recover after that input changes; its earlier failed step is not relabeled as a
terminal failed operation. The zero-binding positive is checked against the
actual signed document. Neither publication outcome proves a member flow.

[Limiter evidence](limiter.json) records exact thresholds, refusal codes,
Retry-After, class/limiter counters and body boundaries. The observation burst
uses invalid `{}` bodies to reach schema refusal without inventing agent
telemetry. Lease requests and usage events use valid product identities. A
single real admitted binding supports the synthetic usage event and duplicate
checks; that does not prove kernel sampling. IPv6 /64 behavior is component
verified; the live connection uses assigned IPv4 loopback ports.

The [Follow-up 3 records](followup3-limiter.json) retain the resolved login
measurement and old assumptions. #145 corrected route classification; #141
introduced shared IPv6 /64 keys. [Attempt history](followup4-attempts.json)
retains two interrupted harness runs before the final fixture was ready.
Private subprocess logs remain mode 0600; the dev-stack startup subprocess uses
umask 022 so copied public CoreDNS files remain readable by its non-root process.

[PITR](pitr.json), its source images/timestamps and [Follow-up 3 compatibility
check](pitr-compatibility.json) remain historical. Follow-up 4 makes no new
physical restore or current-schema compatibility assertion. The full `at-secops`
target includes physical PITR; `at-secops-pitr` runs it independently on a fresh
own-prefix stack. It never restores the live source or shared admin database.

[Secret scanning](secret-scan.json) inventories all source and extra native-peer
container streams, produced node/member logs and audit rows. Raw bodies,
credentials and backup data stay private. [Requests](requests.md) name the
remaining owners and exact prerequisites; [operator procedures](operator-runs.md)
remain unrun. JSON records retain build/image digests, policy/incarnation, UTC,
commands and expected/actual results. Supporting tests do not promote a blocked
full case to passed.

The isolated `at-secops-native-scale` recheck admitted 50 devices and stopped at
native login attempt 51. [Gateway diagnostics](native-diagnostics.json) identify
the configured 256-connection budget; [cohort evidence](native-cohort.json)
records 50 current bindings. This is a lab precondition failure, not a measured
candidate-hardware capacity number. The gateway also logged one transient
device-flow query value; the public scan reports only its category/count.
See SECOPS-10 and SECOPS-16. No 60-device success is claimed.
