# Follow-up 22 — merged evidence through #356

Fixed `main` cutoff **45209609d33d567c5f04d49441799cdd3efac59b**, through
**#356**, checked 2026-10-31 EDT. This cut covers #352–#356. Devstack 22,
members 12, isolation 11, updates 12 and netlab 22 were not merged when this
cut started; their results are excluded. Handover ran no new live acceptance.

**D-098 / D-109 / D-132:** the [five exact roadmap rows](../../tests/handover/testdata/followup22-main.json)
pin current owner measurements. The [snapshot inventory](../../tests/handover/testdata/followup22-snapshots.json)
retains exact dated hand-off prose and frozen owned excerpts. Raw evidence
reads use UUID run directories or owner archives. Members 11's F11 roots,
isolation 10's summary, netlab 21's evidence root and agent-core 29's current
JSON are never claim-test inputs. Provenance paths in copied prose are not
instructions to read or pin those living roots.

## Devstack 21 — #352

The [dated hand-off](../e2e/runs/2026-10-31/followup21/README.md) and D-102
answers cover members 11 / updates 11 / dns-publish 9. Follow-up 20's
**114 files** are archived byte-identically in
[followup20-archive.sha256](../e2e/followup20-archive.sha256).

**Updates F11-01 fixed in the shared installer spec:** `installerSpec.AllowLoopback`
and the exact local `ControlURL` gate add `--allow-loopback` only for the
existing TEST ONLY proxy, default false elsewhere. Fresh run **33b9621a**
passes **3 / 0 / 0**: install exit **0**, generated `result_url` read back
unchanged, certificate-free POST **403**, authenticated invalid body **400**.
This verifies the authenticated endpoint and unchanged config; updates 11's
independent automatic-delivery **27 / 27** stands. The first installer attempt's
origin-slot **.24 collision** with the third DNS node is retained; the successful
run uses its own fresh stack.

**Dns-publish 9 request 1:** the shared trace records a request start before
dispatch, retains outstanding requests, and joins late completions by identity
across the whole closed-case sequence. It adopts **BACKEND §2.1.5**, including
fetch-versus-fallback attribution and completion times. The **4 KiB** fixture
cache is explicit, distinct from the pin's **4 MiB** default. Final run
**cba88bbc** passes **13 / 0 / 0**, with **36 / 36** complete query traces,
**85** distinct authority requests and **70** identities retained across
snapshots. This validates the export; it **does not reproduce F9-INFLIGHT** or
reconstruct historical F10-03. The initial DNS run's incomplete restart trace
remains retained.

The static [members analysis](../e2e/scenarios/followup21-members.md), with
`docs/e2e/followup21-members.json` as its original extract, bounds **F11-01**
to after inner member egress and before observable inner node ingress; encrypted
sub-hops were not captured. **F11-03** retains twelve generation-13 Applies:
all eight failures exceed the **100 ms** guarded install bound, have
`install_ok=false`, no guard release and no verification. This establishes
the failure stage, not its cause. Historical raw command errors, pressure and
injection timing are missing. **F18-01** is the intended **D-125 negative
1 / 1 / 2**; its historical failed candidate-success row stays failed.

Proof names on main: `TestENR02_InstallerFixtureLoopbackOptInScope`,
`TestE2E_Followup21Installer`, `TestE2E_Followup21DNS`,
`TestAT20_EarlierAuthorityCompletionAcrossQueries`,
`TestAT20_OutstandingLifecycleAndMissingReceipts` and
`TestAT20_RestartRetainsStreamsAndPendingRequest`.

## Installer 8 — #353

[Field-by-field reconciliation](../node/install-followup8-2026-10-31.md):
**no installer behaviour change**. `renderUpdater` and same-release reporting
repair match updater 18 (#342) and updater 19 (#351): exact origin, enrollment
server CA, enrolled certificate/key paths and explicit TEST ONLY loopback.
The dated note's unmerged devstack caveat describes its starting base; #352
has delivered that wrapper at this cutoff.

`TestUPD05_RenderedReportingConfigShape` in
`packaging/installer/followup8_test.go` uses literal JSON independently of
updater tags: CGNAT / loopback initial install, stale reporting repair and
identical rerun. It reads no living AT evidence. The consumer proof remains
`TestUPD05_InstallerReportingConfigShape`. Archive wording now covers initial
and same-release retention, with cost **per distinct retained archive**, linked
to the updater's Offline installation paragraph. **F10-01 CLOSED** by devstack
20 (#343) and updates 11 (#346); installer privileged suite **13 / 13**.
Distinct-endpoint CA / private-mirror deployments, physical arm64, power loss
and production rollout remain unmeasured.

## Handover 21 — #354, frozen

The [previous cut](follow-up-21.md) covers #345–#351 at cutoff **#351**, main
`6d1e320d`: `TestHandover_Followup21Claims` retains **21 subtests / 7 pinned
rows / 13 immutable snapshots / 350 owner-archive files**. Its remaining live
register and entrypoint reads now address exact frozen excerpts; assertions,
original source inventories and earlier redirected reads remain unchanged.

## Netlab 21 — #355

The [dated assessment](../poc/G2-followup21-2026-10-31.md) first archives
**82 files byte-identically** under
[G2-followup20-evidence](../poc/G2-followup20-evidence/archive.json). Its own
`docs/poc/G2-followup21-evidence/` remains a living root until netlab 22 archives
it: current measurements below are pinned by the #355 roadmap row and dated
prose, never by reading that root.

**F8-02 captured on the IPv6 underlay** in the privileged fixture:
**10,010 / 64,064 bytes in 12,012.403 ms**. All **19** reverse outer IPv6
packets of **1,360 bytes** are absent at the member; **36 × 1,152**, **one 928**
and **eleven 144**-byte packets arrive. All four captures report **zero kernel
drops**, with **no ICMPv6 PTB**. The MTU-1500 control passes **64,064 bytes in
4.479 ms**. `TestF21_AT09_IPv6Underlay` proves this fixture's observations.

**One mechanism explains both underlays:** the IPv4 **1,340-byte** case becomes
**1,360** with IPv6's **20 extra header bytes**. The oversized reverse datagram
leaves node and tunnel egress and disappears before member underlay ingress,
beyond enforcement. Receive-link MTU rejection is the supported inference;
no kernel drop-reason trace was taken. This establishes the common mechanism
in the captured fixtures, not the exact packets of members 11's uncaptured
native row. Native capture remains devstack 22's item. No global MTU / MSS fix
or IPv6 overlay-service acceptance is invented.

**D-134 enforcement assessment:** an intentional **100 ms** installation
reserve plus **1 ms** truncation gives conservative transition window
**[D−101 ms, D]**. Every pre-D loss stays counted; literal zero pre-D loss is
false. Netlab's own D-104 sample is **26 / 7**, last positive **D−100.730 ms**;
isolation 10's independent sample remains **161 / 41**. The outer guard is
the measured atomic mechanism; completed reload distributions are:

| Population (101 completed guards) | p50 ms | p90 ms | p99 ms | max ms |
|---|---:|---:|---:|---:|
| Outer guard | 32.480 | 41.400 | 45.396 | 52.675 |
| Install | 16.698 | 21.783 | 28.473 | 30.818 |

The historical **52.596 ms** maximum is not a ceiling. An install overrun
never opens the guard: rollback atomically replaces it with closed
generation-0 rules. D-104 / D-120 / AT13 / AT29 predicates are unchanged;
no tolerance or new acceptance pass is introduced.

**F11-03 rollback:** three injected verifier-read faults with an explicit
**200 ms hold** give baseline→recovery brackets **304.815 / 280.395 / 285.297 ms**,
while the failed-Apply guards are **34.4 / 28.3 / 24.0 ms**. The real rollback
closes the admitted sibling until the next successful Apply.
`TestF21_AT02_RollbackSibling` retains **397 ingresses / 224 egresses / 173
losses**, including **150 outside** all completed guards. These diagnostic
durations are not historical F11-03 outage durations or a recovery guarantee.
For **F11-01**, the first unconfirmed interval remains member tunnel egress →
node ingress; the native AT01 route is outside this fixture.

## Agent-core 29 — #356

[Controlled measurements](../node/followup29-install-contention-2026-10-31.md)
**do NOT reproduce the install overrun**. Same-source quiet / loaded / quiet /
loaded live + agent suites and six two-source interventions use host / container
busy-loop workers up to **384 on 24 logical CPUs at 0 % idle**. All **40 loaded
Applies per intervention** complete without a 100 ms overrun; maximum
render→installed is **53.133 ms**. The six reproduction diagnostics remain
negative reproductions, not passing tests. The **184.078 ms outer guard** is
a different measurement from the install budget.

Earlier overruns correlated with chain-gate load (follow-up 28's **152 ms**,
isolation 10's **49 failed installs**, members 11's generation-13 Applies);
this is **not a causal conclusion**. The historical cause stays **unproven**.
Loaded meter / restart / expiry failures remain failed and undiagnosed.

Additive, opt-in **TEST ONLY host context** in `node/agent/measurement_host.go`
brackets every Apply with before / after samples and sanitized returned command
status. Sampling occurs **outside the measured Apply operation**. Unavailable
samples stay explicit; missing command status is unknown. Existing rows,
sequence rules and **D-106 digest inputs are unchanged**. The producer audit
validates **1,874 Apply host brackets** (#356). Proofs:
`TestF29_HostSampleAccountingAndUnavailable` and
`TestF29_ApplyHostContextIsOptInAndDoesNotChangeTiming`.

**Rollback is the node-wide generation-0 baseline by design, not the previous
generation.** A failed Apply closes every member, including the admitted
sibling, until the next successful Apply. Repeated failures can keep baseline
indefinitely; the **100 ms bound is an install budget, not an outage ceiling**.
Revocation remains preserved. S2 records a real sibling baseline drop after a
successful Apply and successful live verifier, followed by
`member_path_unconfigured`: not every `verification_HEALTH_CHECK_FAILED`
baseline means nft / tc verification failed. S2 is not the historical F11-03
mechanism. **AT02's admitted positive control stays failed at this cutoff**.

No **D-133 A/B** was invoked: no Apply / rollback ordering, timeout, policy or
production forwarding change. Follow-up 28's four Sources links now target
the byte-identical `history/follow-up-9/` archive. Three requests are routed:
executor command-lifecycle evidence → netlab 22; sibling availability contract
→ D-135; consumer labels separating failure stage / host context / unproven
cause → members 12 / isolation 11.

## Decisions, findings and owner boundaries

The [decisions and findings register](README.md#follow-up-22-decisions-register-2026-10-31-edt)
adds **D-134 (16:0x EDT)** and **D-135 (17:2x EDT, provisional — pending the
owner's confirmation)**. D-134 records expiry reserve and measured guard cost,
with every pre-D and renewal loss counted. D-135 proposes the AT02 positive
control report: revoked source dropped; admitted sibling outage bracketed
baseline→recovery, ending at the next successful Apply, every loss counted.
The bracket is **reported, never waived**. Any stronger availability guarantee
requires a separate enforcement design under D-133. Owner confirmation of
D-135 is still required; no historical failed result becomes a pass.

Closed: updates **F11-01** (#352 fix + measurement), installer-side updates
**F10-01** (#353). Updated: **F8-02** common mechanism, native capture pending;
members **F11-01** transport interval, route correlation pending; members
**F11-03** install stage established / rollback measured / cause unproven,
expectation rewrite under provisional D-135 pending members 12; install-overrun
cause **unproven** (#356).

Unchanged: **F9-INFLIGHT**; members **F11-02** readiness gate pending members 12;
**F18-01** intended D-125 negative (now explicitly documented by #352);
**ISO-F6-03** writer unattributed; **Dependabot #22** open for owner disposition;
production **blocked_response_ttl** owner value. Every historical failure stays
failed and unattributed. Physical arm64, power loss, production rollout, native
platform routes, full fault matrices and QUIC / PMTU acceptance remain open.

Requirement scope: PKG01 / ENR02 / ENR03 / ENR04 / UPD05 (§6.1–§6.3,
§13.10.3, AT05/AT07/AT23–AT27); DNS02 / SEC02 (§12, §16.2, AT20);
EXIT02 / APP01 / NFR01 (§8.2–§8.3, AT01/AT09); MEM03 / ISO02 / ENT02 /
SEC01 (§8.2, §10.3, §11.5, §16.3, AT02/AT13/AT29). This is evidence
reconciliation, not expanded §19 acceptance or a production guarantee.

### Requests to orchestrator

2026-10-31: **Owner decision — confirm provisional D-135**, including the
baseline→next-successful-Apply admitted-sibling bracket and counted-loss
reporting. Any stronger availability guarantee needs a separate D-133 design.
Members 12's expectation rewrite remains pending that decision. Existing
netlab 22 executor evidence, devstack 22 native transport and members 12 /
isolation 11 consumer work stay with those lanes. No outside-owned file was
changed or new acceptance predicate installed here.
