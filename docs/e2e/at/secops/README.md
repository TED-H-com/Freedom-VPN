# Wave 4 security and operations acceptance

Follow-up 2 uses the shared dev stack with
`FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000`.

```sh
make -f mk/at-secops.mk at-secops
```

The runner refuses an occupied namespace, creates fresh TEST ONLY state, runs
live probes and a network-none physical PITR, removes its Docker objects, then
runs supporting suites with `-p 1 -parallel 2`. Exit 2 preserves blocked
acceptance. It never restores the live stack or shared admin database.

| Case | Evidence | Scope |
|---|---|---|
| AT28 | [record](AT28.json), [live](live.json) | Sessions, native-device ownership, role refusals, two node certificates, expired/revoked TLS, replay, CSRF/ETag/limits and stream scan |
| AT29 | [record](AT29.json) | Requires serving=1 and increasing admission counters before deadline faults; a publication operation alone cannot satisfy the gate |
| AT30 | [record](AT30.json), [physical PITR](pitr.json), [procedure](pitr.md) | pgBackRest base backup/WAL, target boundary across ledger/grant/audit, isolated replay without extension or double counting. Actual node archive and external reconciliation remain unmeasured |
| AT31 | [record](AT31.json), [dependency check](dependencies.json) | 20 API grants and one native device; full capacity blocked without counted serving flow and 60-peer hook. Prior login burst is historical pending D-060 |
| AT32 | [record](AT32.json), [review](scope-review.json) | DOC01–DOC13 corrections retained; physical source/platform comparison needs operator runs |

D-050: every live fixture input uses an API or exported lab helper. Groups,
fair-use/DNS/plan policy revisions and activation use admin APIs. Grants have
finite ends; entitlement projection is polled. Native NetBird user admission
uses embedded Dex and production discovery. `e2e.PrepareServingBaseline` supplies
signed release/support inputs. **No live fixture SQL writes remain.** The separate
PKI component test retains its isolated dbtest identity rows for authentication
against an inert handler; these are not serving evidence.

The synthetic ledger reads a real admitted binding. Its original mTLS uploads
are retained by the test sender. Accepted usage is not evidence of kernel
sampling or forwarding. Capacity measurements require a counted serving flow;
[historical API diagnostics](capacity-history.json) and the [Follow-up 1 login
burst](followup1-login.json) preserve their original scope and source.
The instructed dependency check found neither control follow-up 10 nor devstack
follow-up 7 on origin/main. No merge, limiter tuning or recovery workaround was
performed. The 41st-request Retry-After remeasurement stays blocked on D-060.

For the independent physical half on a **fresh** own-prefix running lab:

```sh
FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-secops.mk at-secops-pitr
FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 make -f mk/e2e.mk e2e-down
```

This uses the repository backup scripts, not a second backup implementation.
`at-secops-pitr` removes its restore container/volume and leaves the source stack
for inspection; the final command removes that source. Never invoke it against
an occupied instance started by another operator. The old `logical-followup1.json` is
historical logical-only evidence and is not used for current verdicts.

[Secret scan](secret-scan.json) covers component output, all twelve source
container streams, produced node/member logs and audit rows. `pitr.json` records
the separate restore-stream scan. Credentials, raw bodies, backup data and logs
stay in `.agent/secops`. [Requests](requests.md) name owners and prerequisites;
[operator procedures](operator-runs.md) remain unrun. Each AT JSON contains
build/image digests, policy/incarnation, UTC, commands and expected/actual results.
Supporting checks never promote a blocked full case to passed.
