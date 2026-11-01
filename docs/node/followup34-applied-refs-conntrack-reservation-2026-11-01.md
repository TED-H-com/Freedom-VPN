# Follow-up 34: applied observation provenance and conntrack reservations

Production keeps option (c), as directed by provisional D-141. This change
addresses r19 M-2 and L-2. The follow-up 33 design and test-only reference stay
as history; no intersection-serving implementation is added.

## M-2 is reachable

The ordinary Step accepts desired h, observes a generation-stale lease g,
baselines, ACKs h, renews and applies h before building its next report. That
ordinary path does not send the review's mixed serving report. It is not an
invariant of all production paths:

1. A verified generation g is serving. Same-boot adoption legitimately schedules
   a renewal; renewal can also become due through the normal deadline/lead.
2. Step accepts desired h. Its second `syncLease` can obtain a lease for h.
   The central issuer signs `nodes.desired_config_generation`, independently of
   the request ACK (`internal/leases/leases.go`, `RenewTx`, the generation comment
   immediately above construction of the signed lease). A ready, published node
   can receive member grants on this renewal.
3. `evaluate` permits the h candidate. `enforce` attempts the shared node lock
   three times at 50 ms intervals. `ErrBusy` retains the previously verified
   g rules and `e.serving=true`; it does not baseline or ACK h.
4. `renewAppliedLease` cannot issue a second request in that Step because
   `leaseAttempted` is already true. Its generation comparison returns without
   applying. The end-of-Step `observe` then builds/enqueues the report, and
   `flush` can send it. Making the observation due requires only normal cadence.
   The final live read still verifies the retained g model.

`TestF34_ReportBeforeNewApplyOnBusyLock` executes this entire Step, including
same-boot adoption, signed desired/lease responses, lock contention and delivery
to the fake receiver. It passes on unchanged main and the candidate, with ACK g,
desired/lease h, forwarding true, no intervening baseline or Apply, and the old
prefix still in the installed model. This is a provenance fix; it does not
change the existing lock-contention forwarding behavior.

Every production report call is accounted for: `observe` is the sole caller of
`report`. Step calls `observe` on revocation after withdrawal, after a rejected
initial live read, and after enforcement/renewal. `repairGeneration` queues its
failed read after setting serving false. Deferred `finishServingStep` observes
when final verification fails, the stale-hold state changes, or a frozen identity
needs a report. Ordinary successful final verification only writes a heartbeat.
The earlier Step `flush` replays immutable outbox bodies before desired sync;
it does not rebuild them with the new document. Engine Step is single-threaded.

## Applied provenance

The ACK now has a small retained snapshot of its config generation, binding
revision and policy refs. `ackApplied` records the snapshot and applied generation
in one SQLite transaction, after the completed Apply or baseline. `report` takes
both refs from this snapshot, checked against the ACK generation, rather than
from the current desired input. Neither the snapshot nor its presence authorizes
forwarding, restart adoption or a lease.

The metadata uses the existing table; there is no schema migration. Restart
loads the snapshot independently of the newest desired document. Upgrading a
legacy store can reconstruct it only from an authenticated desired document with
the exact applied generation. If historical refs are missing or mismatched, the
ACK remains available but its two provenance fields are null. Frozen/uncertain
identities still suppress all three ACK fields. A failed atomic write preserves
the old ACK and snapshot together.

Read-only inspection of `internal/nodes` found no production classification
reading `AppliedBindingRevision` / `AppliedPolicyRefs` or their JSON names.
They remain part of the stored observation body. Serving readiness compares
applied/desired generations (`transitions.go:129`); publication evidence uses
those generations, checks, forwarding and a persisted matching lease
(`publication_evidence.go:38`). Wire-nodes 34's hold corroboration joins the
reported lease ID/node/incarnation to its own stored signed generation
(`generation_hold.go:38`), never to these two fields. No central reader change
is needed for this fix.

## L-2 reservation and failure semantics

Before a nonempty revoke batch enters Apply, one FULL-synchronous SQLite WAL
transaction stores the next attempt count and all pending address identities.
Only after that commit succeeds can Apply install rules and run `conntrack -D`.
The count is no longer incremented after Apply. Partial failure retains only the
failed identities, preserving the count. A generation change cannot reset it
while addresses remain pending. Successful completion of all pending deletes
clears it; restart loads it and enforces the existing limit of twelve.

The limit counts reserved serving Apply batches, not individual `-s` / `-d`
commands and not elapsed seconds. A crash before the first delete, or an Apply
failure before reaching it, conservatively consumes a slot. That uncertainty
cannot be refunded safely from a generic Apply error. Exhaustion baselines;
existing baseline-only cleanup can retry and clear the block after successful
cleanup. No forwarding is authorized by those cleanup attempts.

A failed reservation withdraws and never enters Apply. Reporting distinguishes
an uncompleted reservation from a verified serving ruleset. The older invalid-
revoke test retains its assertions that no delete runs and forwarding is off;
its counter/pending assertions now expect the conservative reservation.

The converted r19 crash proof runs the real `enforce.Apply` delete loop with a
fake executor, interrupts immediately after its first delete command, and reads
the count/pending identities through a separately opened database connection,
without running post-Apply persistence. Twelve interrupted batches remain
counted, and a thirteenth serving Apply is refused. This is deterministic fault
injection at the production boundary, not a host/kernel crash experiment.
Atomic-write failure, restart, successful clear and failed pre-Apply storage are
also covered.

## Latency and acceptance boundaries

The reservation precedes Apply's first instruction, so it also precedes the
ruleset install. It adds no work inside the render/install/release guard and
does not change D-104's 100 ms bound or 1 ms granularity. Its time must still be
counted in receipt-to-revocation and baseline-to-recovery elapsed time.
On the ordinary stale-lease option-(c) path, the initial baseline already stops
the revoked source before this reservation; the new work extends the later
recovery path. If a matching newer lease permits direct Apply, the write is
before the first restricting install. Existing kernel deadlines continue to
expire independently, and Apply advances the monotonic model before rendering.

A disk-backed 240-iteration measurement per batch size on the development host
measured the reservation itself (milliseconds, nearest rank):

| Pending addresses | p50 | p90 | p99 | max |
| --- | ---: | ---: | ---: | ---: |
| 1 | 1.079555 | 1.095849 | 1.589684 | 2.070340 |
| 64 | 1.070564 | 1.245404 | 1.546023 | 1.583537 |

This is additive pre-Apply work, not an AT pass or a quiet-host kernel comparison.
The same synchronous store configuration is used in production. The benchmark
has no production reset or timing switch. Reproduce with disk TMPDIR/GOTMPDIR:

```sh
go test ./node/agent -run '^$' -bench '^BenchmarkF34_ConntrackReservation$' -benchtime=240x -count=1
```

The affected acceptance consumers are members AT02's revoked-device cutoff and
unchanged-sibling baseline-to-Apply recovery, AT04 member suspension where it
removes pending addresses, isolation AT13 established-flow revocation/restart,
and the isolation D-104 renewal/whole-node-expiry row if a measured Step also
has pending revocations. Ordinary expiry and renewal with no queued revoke do
not execute this new write. The assembled `TestLabD087_D104_GrantRenewalExpiryCorrelation`
and the live mechanism gate are separate evidence; neither rewrites historical
members/isolation failures. Physical arm64 remains unmeasured; both architectures
are cross-built.

## Same-host D-133 comparison, judged under D-138

The first A1,B1,A2,B2 sequence ran both complete suites per leg. An additional live-only R-A1,R-B1,R-A2,R-B2 sequence retained the same frozen sources/settings to obtain a live A comparator after three first-window live exclusions. All twelve tests passed. Six runs qualify at entry and exit; six remain excluded. The experiment ran 13:17:40–14:32:22 UTC (74m41s), within the 90-minute limit, with 48 preflight samples, 36 rejected. There is no blocked/unrun leg. The final R-B2 test passed but its exit sample failed r<=3; it is not pooled.

Preflights ran every 60 seconds: no competing `gates.sh|mk/at-|go test`, and six current `vmstat 5 7` intervals all r<=3/id>=80 (the since-boot row is excluded as in D-133). Exit samples apply the same criteria. Every load average, D-state count, vmstat range, rejected preflight and exclusion is retained in [the JSON companion](followup34-comparison-2026-11-01.json); load average was not a criterion. Qualification here is at suite entry/exit: no separate host vmstat receipt immediately before in-suite traffic was captured. These are bounded mechanism-suite comparisons, not full D-133 acceptance of paired end-to-end revocation/sibling outages or D-141's deferred 80-generation intersection experiment.

A=335047a2b650db1d64bb4e0c59fc042fad74da51, B=30aa9aa36a04ada82c341fadd2a490f34ad19990. The public evidence pins their compiled agent binaries, operation streams and fixture image. All live image builds have the same manifest `sha256:cc7ef55542982ffae720e29fe1781ebd618514227157014d4c0a420a7b006fdc` and config `sha256:07f8bc41768328a148d181a025794457ae38b2df5f0f9a8b7e616b9b89035fb1`. Both arms use the same host/kernel, fixture code, test-only observation, probe cadence and runtime settings. Test identities/keys are freshly generated; their bytes are not claimed identical across runs.

| Leg | Suite | Test | Entry/exit quiet | Completed guards | Outer guard p50/p90/p99/max (ms) | Failed Applies |
| --- | --- | --- | --- | ---: | --- | ---: |
| A1 | agent | PASS | excluded | 134 | 31.613148/39.581193/47.969945/48.193744 | 2 |
| A1 | live | PASS | excluded | 117 | 28.781569/39.360529/45.369663/53.844989 | 4 |
| B1 | agent | PASS | yes | 134 | 28.718384/38.271980/45.512533/46.536238 | 2 |
| B1 | live | PASS | yes | 117 | 27.993203/37.533457/44.289492/47.052861 | 4 |
| A2 | agent | PASS | yes | 136 | 29.933546/39.347551/46.293936/48.050159 | 2 |
| A2 | live | PASS | excluded | 117 | 27.507083/35.814034/44.979069/45.667389 | 4 |
| B2 | agent | PASS | yes | 133 | 28.981856/39.759195/44.724428/50.593645 | 2 |
| B2 | live | PASS | excluded | 117 | 30.975977/39.621804/45.680191/53.155626 | 4 |
| R-A1 | live | PASS | excluded | 117 | 26.967743/37.382201/41.607509/42.872458 | 4 |
| R-B1 | live | PASS | yes | 117 | 27.970299/36.739465/43.736890/46.355343 | 4 |
| R-A2 | live | PASS | yes | 117 | 30.170810/39.030157/44.920442/51.818585 | 4 |
| R-B2 | live | PASS | excluded | 117 | 28.826929/36.433855/45.800662/49.046024 | 4 |

Qualified per-arm pools (every eligible leg across windows; nearest rank):

| Pool | Legs | Guards | Outer guard p50/p90/p99/max (ms) | Install command p50/p90/p99/max (ms) |
| --- | --- | ---: | --- | --- |
| A-agent | A2 | 136 | 29.933546/39.347551/46.293936/48.050159 | 15.556153/22.308913/25.965947/27.369166 |
| A-live | R-A2 | 117 | 30.170810/39.030157/44.920442/51.818585 | 15.140917/21.545103/26.203211/29.416583 |
| B-agent | B1, B2 | 267 | 28.875659/39.027046/45.512533/50.593645 | 14.943778/21.172240/28.819306/29.597273 |
| B-live | B1, R-B1 | 234 | 27.970299/36.968636/44.289492/47.052861 | 14.329811/19.914038/26.303544/27.271589 |

Live renewal stream (each has 76,000 sent packets; losses are counted even on excluded legs):

| Leg | Quiet | Lost | Loss ingress inside/outside guard / missing | Last positive relative to D (ms) | Revoke pre-Apply brackets (ms) |
| --- | --- | ---: | --- | ---: | --- |
| A1 | excluded | 1144 | 1144/0/0 | -100.657078 | 0.019295, 0.018094 |
| B1 | yes | 1062 | 1062/0/0 | -100.509613 | 1.112091, 1.165807 |
| A2 | excluded | 995 | 995/0/0 | -100.606861 | 0.019020, 0.012461 |
| B2 | excluded | 1139 | 1139/0/0 | -100.709849 | 1.053328, 1.157105 |
| R-A1 | excluded | 1037 | 1037/0/0 | -100.752063 | 0.018759, 0.028129 |
| R-B1 | yes | 1057 | 1057/0/0 | -100.685154 | 1.135230, 1.029201 |
| R-A2 | yes | 1126 | 1126/0/0 | -100.835608 | 0.021953, 0.024833 |
| R-B2 | excluded | 1124 | 1124/0/0 | -100.570588 | 1.019888, 1.015309 |

The retained failed Applies are expected injected pre-install failures (two per agent run, four per live run); none reached the install command. Full incomplete timings and failed live reads remain in the evidence rather than entering the completed-guard distribution. No test failed on either arm. No loss is waived, including all losses on excluded legs; the live renewal stream's guard overlap is a timestamp correlation, not a causal diagnosis.

The existing `host.before_apply.end_ns` → `timing.start_ns` bracket contains the reservation and small common setup. Eligible A revoke samples are 0.021953/0.024833ms; eligible B samples are 1.112091/1.165807/1.135230/1.029201ms. That agrees with the direct write benchmark and the code's placement before the restricting install. It is additive work outside the D-104 render/install/guard interval; a low guard percentile cannot hide it. Absolute install-start joins and full run/boot/sequence identities are in the JSON companion.

D-138 verdict: pool every eligible leg across windows, keep exclusions, and do not use the withdrawn two-leg envelope rule. Agent outer-guard p50/p90/p99 are lower on pooled B, but max is higher; install-command p99/max are higher. Live pooled B is lower on all four outer-guard statistics, but the first window has no eligible A live leg and the second has only one. This does not establish a repeated direction in each window or a code explanation for a broad guard/packet-loss shift. There is no B-only test failure, and no attributable system-wide timing improvement or regression is asserted. The reservation's measured pre-Apply write cost is explicit. Full paired receipt-to-cutoff/recovery AT evidence remains with the dated members/isolation requests; no threshold or historical acceptance status changes.

## Review-proof scope

The adopted agent proof file retains all eleven tests that passed unchanged main,
byte-for-byte within each test body, including the reference tests that already
passed. Only M-2 and L-2 are converted. Both converted tests fail against the
unchanged-main production sources and pass with the fixes. No watchdog review
file is adopted.

The eight removed design-only TODO proofs are
`TestReviewR19_Q1_ReferenceReplayExtendsDeadline`,
`TestReviewR19_Q2_ReferenceAdmitsUnboundedCeil`,
`TestReviewR19_Q2_ReferenceKeepsDeniedResolver`,
`TestReviewR19_Q2_ReferenceIgnoresRateModeAndWitnessSnapshot`,
`TestReviewR19_Q2_RequiresTCKeepsCandidate`,
`TestReviewR19_Q3_UnsupportedObservationDoesNotBaseline`,
`TestReviewR19_Q4_QualifyingShareOnMergedEvidence`, and
`TestReviewR19_Q5_RollbackSourceDoesNotBaseline`.
Their unused `nearestRank` helper is removed too. The original proofs remain on
the review branch and under D-141's reconsideration conditions.
