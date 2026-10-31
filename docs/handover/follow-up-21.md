# Follow-up 21 — evidence through #351

Fixed main cutoff **6d1e320dd7dee7b8a520f116a347e9f6f2d28d8e**, through
**#351**, 2026-10-31 EDT. Handover ran no new live acceptance. Devstack 21,
netlab 21 and agent-core 29 were not on main when this cut started; their
results are excluded. No new decision: **D-133 remains the latest**.

The [seven roadmap rows](../../tests/handover/testdata/followup21-main.json)
pin #345–#351. `TestHandover_Followup21Claims` checks these rows, existing Go
test names, owned guidance and [immutable source snapshots](../../tests/handover/testdata/followup21-snapshots.json)
under **D-098 / D-109 / D-132**. Current members 11, isolation 10 and updates 11
root evidence is cited by roadmap row / PR only, never read or hash-pinned by
this claims test. Historical measurements use the owners' `history/` archives.

## Dns-publish 9 — #345

Members F10-03 was investigated through real node/dns Apply on two pinned
**AdGuard Home v0.107.59** resolvers. Live defaults after Apply were
`cache_size: 4194304`, `cache_ttl_min: 0`, `cache_ttl_max: 0`,
`cache_optimistic: false`, `upstream_timeout: 10 s`, `fallback_dns: []`.
Existing operator overrides are retained by Apply; these are measured fresh
defaults, not a claim about every deployed resolver.

- **F9-HIT:** ordinary primary hits retain TTL-10 data with per-second
  decrement and no authority exchange; a fresh fetch occurs after 10.1 s.
- **F9-OTHER:** the secondary has an independent cache.
- **F9-ROUND:** whole-second expiry allows a fresh same-primary answer at
  about 9.35 s; this is precision, not optimistic caching.
- **F9-INFLIGHT:** an older upstream reply arriving after the client abandoned
  its request but before the **10 s upstream timeout** replaces a newer cache
  entry (**new → old within 1.3 s**). This is inherent insertion ordering;
  no exposed cache, mode or timeout setting fixes it. **TTL is a maximum
  lifetime**, not minimum retention or cross-query consistency.

[BACKEND §2.1.5](../dns/BACKEND.md#215-f10-03--positive-cache-and-ttl-measured-2026-10-31)
requires a uniquely identified responder, matching query bounds/cache keys,
actual settings, and complete upstream history including outstanding-request
completion times. An exchange proves a fetch, not fallback. No exchange
supports only an inferred hit; negative-cache state can remain unknown.
Latency alone does not identify the cause. The historical members **F10-03
stays failed / unattributed**; this new→old experiment does not reconstruct
its old→new response. The [frozen backend excerpt](../../tests/handover/testdata/history/follow-up-21/dns9-backend.md)
preserves those limits.

`TestAT20_PositiveCacheTTL` in `tests/netlab/dns/positive_cache_test.go` is
enforced by the mandatory **dns-positive-cache** DNS PR gate in
`.github/workflows/dns.yml`. A changed backend pin must remeasure semantics.
The production **blocked_response_ttl remains the owner's decision** (§12.5);
no TTL or production setting is selected here.

## Updates 11 — #346

Run **daddad24**, product baseline `96ec0630`, consumes devstack 20's fixture
and installer 7's generated reporting **without an adapter**. Installer
checks pass **27 / 27**; **regular AT23–AT27 pass 5 / 5**. AT26 physically
rolls back after the candidate tunnel probe fails, restores last-good with
five passing health checks, and reports rolled_back / needs_attention.
AT27 requires that same-run physical rollback witness; an HTTP correction
stimulus is not its substitute. See `docs/e2e/at/updates/follow-up-11.md`
and the #346 roadmap row; `TestUpdatesFollowup` retains the campaign.

The **artifact-origin question is closed by measurement, no rejection**.
`allowed_artifact_origins` grants **private / CGNAT exceptions only**, not an
exclusive list of allowed hosts. The measured non-private release origin
needs no exception; cached hand-off and deliberately uncached seed both
succeed without a policy or hosting change. No private-mirror deployment
claim follows.

The shared wrapper exits **64** without the **TEST ONLY --allow-loopback**
opt-in. Generated `result_url` is the authenticated split-listener proxy;
server trust, enrolled identity, exact control-origin exception and whole
config hash are unchanged. The route capture records **13 result POSTs**:
**11 stimuli + 1 automatic success + 1 automatic failure**. Repeat flush adds
none; **result_url_missing never observed**. This is measured transmission in
that interval, not a guarantee against transport retries in all deployments.

Follow-up 10 is archived byte-identically (**39 files**) under
`docs/e2e/at/updates/history/follow-up-10/`. Updates F10-01 closes with #343's
fixture and #346's regular measurement. **F18-01 stays the D-125 negative**:
stopped-producer apply/replay/recover **1 / 1 / 2**, failed seal retained,
forwarding blocked. Updates **F11-01 → devstack 21** (explicit test-loopback
wrapper); updates **F11-02 → updater 19**, now fixed in #351.

## Handover 20 — #347, frozen

The previous cut covers **#338–#344 at cutoff #344**.
`TestHandover_Followup20Claims` retains **18 subtests / 7 pinned rows /
38 immutable inputs** and the D-133 registration. Its [dated report](follow-up-20.md),
source manifests and existing snapshots remain unchanged. Its formerly live
register and entrypoint assertions now read exact owned excerpts in the
[follow-up 21 archive](../../tests/handover/testdata/history/follow-up-21/register20.md),
preserving the original assertions while this cut updates current dispositions.

## Members 11 — #348

Run **7edb263f** on **#343 / #344 / #336** yields **2 pass / 6 blocked / 2 fail**
complete cases (558 / 5 / 3 subchecks). The new F10-02 finite-expiry row
passes **186 / 186**; the new F10-03 cache row passes with **AT20 blocked
81 / 0 / 0**, **39 / 39 traces**, primary TTL 9/no exchange before TTL and
private_ipv4/observed exchange afterward. Both historical failures stay
**failed / unattributed** in the **35-file follow-up 10 archive**.
**F10-04 closed under D-129**: generation-29 recovery qualifies at the
generation yielded by the unchanged sequence; 30 is not a contract value.

**Full AT02 fails 121 / 0 / 2**:

- **Members F11-02:** the D-077 positive probe starts at **generation 10**
  with **zero member grants / AWAITING**. This is a members-readiness gate;
  require populated-grant serving before the positive probe. No producer
  defect or unauthorized forwarding leak is established.
- **Members F11-03:** the admitted sibling hits baseline during
  **generation-13 failed Applies**, with **HEALTH_CHECK_FAILED**. The rollback
  writer is recorded, but the cause remains **unattributed → agent-core 29 /
  devstack 21 / netlab 21**. Later recovery does not repair this failed control.

**Fresh AT09 fails 37 / 5 / 1**: desired / applied **30 → 30**, serving
**6.03 s**, first counted byte **11.40 s**; all **three provisional reports
pass D-126**. The **dual / 1280** row returns **13,013 / 64,064 bytes in
12,010 ms** over a **direct P2P IPv6 underlay**. This is **F8-02 recurring**,
distinct from devstack 20's **IPv4** capture; netlab 21 owns assessment.
The underlay does not change the D-123 IPv6Block overlay policy. Five QUIC /
platform selectors remain blocked; the recovery pass does not pass full AT09.

New **members F11-01**: the selected AT01 route carries **five member packets
with no node / origin receipt → devstack 21**. A node enforcement drop is not
established. `TestATMembers_Live`, `TestAT09_D126CapturedProvisionalSequence`
and `TestAT02_AT20_SharedDiagnosticSerialization` are the named owner checks.
Current results are pinned by #348's roadmap row, not living `F11-*.json`.

## Portal 19 — #349

**Dependabot #22 cannot be closed from the lockfile**: minimatch **< 11.0.1**,
**GHSA-g3hv-75wg-m249**, scope **development**. `npm audit fix` changes
nothing. The only complete override (**^11.0.1 → 11.0.3**) clears the audit,
but after a clean `npm ci` it breaks eslint-plugin-jsx-a11y's nested-label rule
(`jsx-a11y/label-has-associated-control`): the legacy callable export is absent.
Both dependency files stay **byte-identical**.

All four dependency paths are dev tooling: eslint **9.39.5** (direct and via
**@eslint/config-array / @eslint/eslintrc**) and **eslint-plugin-jsx-a11y** use
**3.1.5**; **@typescript-eslint/typescript-estree** uses **9.0.9**.
`docs/portal/DEV.md` records **no production-bundle path**, including the
production dependency tree and source-map inspection. Restored dependencies
pass **portal-verify 74 / 74** and **portal-e2e 9 / 9**. **Owner disposition
pending**; this is a **dependency-hygiene item, not a product defect**. Do not
apply the blanket override or dismiss the alert to claim closure.
[Frozen advisory investigation](../../tests/handover/testdata/history/follow-up-21/portal19-advisory.md).

## Isolation 10 — #350

Run **bfcc3880** on baseline **96ec0630** (through #344) records
**AT10 PASS 9 / 0 / 0 on the real D-087 transition**: wire-nodes 33's seam
through devstack 20's export, **false → natural signed empty lease → HTTP 202
ready / passed observation with forwarding false and awaiting_member_grants
→ true → distinct serving receipt joined to a new populated signed lease
→ counted native forwarding**. True → accepted serving is **962.448 ms**;
**generation 13** and all **297 wire hashes** remain byte-identical.
Historical **4,767 / 5,021 / 5,025 ms** stand as separate samples.
`TestD087_OriginalWireAndAcceptedTransition` checks the original wire/joins.

**AT12 PASS 10 / 0 / 0**. **AT13 BLOCKED 6 / 1 / 0**: the fresh whole-node
**D-104 row passes**, **161 positives / 41 drops**, last positive
**D−99.655 ms**, first drop **D−63.565 ms**, with **one pre-D ingress loss
counted**. Literal zero pre-D loss is false. The historical **ISO-F6-03
failure (96 / 65 / 41) stays failed**, its writer **unattributed**. New restart
proofs cannot retrospectively identify it. **AT11 BLOCKED 2 / 1 / 0**.

Renewal retains **1,396 probes / 30 losses**, all inside completed command
guards; every loss stays counted. **D-120 unchanged** (100 reloads, one
eligible loss), **ISO-F6-01 6 / 6**, **D-106 654 / 654**.
`TestAT13_D104StrictBoundaryAndMissingEvidence` retains the strict negative.

| Completed guards (968), ms | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Outer | 31.09 | 40.53 | 48.85 | 52.93 |
| Install | 16.11 | 22.32 | 27.70 | 30.62 |

This is an **independent sample after agent-core 28's A/B**, not a causal A/B
comparison. Two earlier attempts are retained separately. The first has
**49 failed installs**, including **Apply 712 at 215.891 ms**, during the
**14:1x chain-gate load → agent-core 29 under D-133**; host contention is not
established as its cause. The second fails the fixed-generation predicate
after counted traffic triggers projection **12 → 13**. Follow-up 9 is archived
**86 / 86**. Both failed attempts and the historical failure remain immutable.
Current measurements are pinned by #350's roadmap row; the test never reads
the living isolation summary.

## Updater 19 — #351

`docs/updates/node-followup19-2026-10-31.md` consumes updates 11's generated
reporting measurement without an adapter. Its [frozen hand-off](../../tests/handover/testdata/history/follow-up-21/updater19.md)
keeps the unmeasured remainder explicit: **physical arm64, power loss, fleet
rollout, production central, worst-case 30 s / 100 ms bounds**, plus the named
fault injections that remain component coverage only.

**TestUPD05_InstallerReportingConfigShape** proves an intact installer-shaped
config builds the **HTTPSink** with exact origin policy, enrolled identity and
a distinct **endpoint-server CA**. Removing applicable CGNAT or TEST ONLY
loopback permission fails construction. **result_url_missing is unreachable
there**; the absent-result_url pending/repair negative remains unchanged.
No reporting-field divergence was found.

**Updates F11-02 fixed** in `docs/updates/node-updater.md`: initial and
same-release installs retain the **verified archive** and envelope. Only
**absence chooses the guarded signed-URL download**. Corrupt/unreadable
archives fail closed rather than downloading a replacement. This corrects
documentation; it adds no runtime behavior. The **D-132 audit has zero
docs/e2e reads** across 37 updater + 8 updatejournal test files.

## Findings and remaining work at #351

The [current findings register](README.md#findings-register-at-the-follow-up-21-cutoff)
closes **F10-04 (D-129, #348), updates F10-01 (#343 + #346 measurement), the
artifact-origin question (#346, by measurement, no exception needed), updates
F11-02 (#351), and assembled D-087 AT10 (#350)**. Opened: **F9-INFLIGHT**
(inherent, documented), **members F11-01 / F11-02 / F11-03**, and the **F8-02
IPv6 recurrence**. Members and updates finding numbers are separate namespaces.

Still open at this cutoff:

- ISO-F6-03 historical writer; F8-02 on **both underlays** (netlab 21).
- Members F11-01 / F11-02 / F11-03; updates F11-01 (devstack 21).
- F18-01 (devstack 21); Dependabot #22 owner disposition.
- The **two D-133 measured limits**: literal zero pre-D loss and the historical
  52.596 ms outer ceiling; netlab 21 assessment, no tolerance requested.
- Production **blocked_response_ttl**; **commercial feed approval** (§12.3).
- **AT11 real platform routes**, **AT13 full fault matrix including physical
  arm64**, **AT29 / AT31 resources**, real platform clients and **QUIC / PMTU
  tooling**. Physical arm64, OS reboot, power loss and fleet deployment remain
  separate measurements; passing component or lab subsets do not close them.

Operator guidance changes only for the positive-cache limitation, corrected
retained-archive installation behavior, and Dependabot #22 status. The
[bilingual member notice](member-guide.md) explains the corresponding limits
without asking members to change security settings. No new production value,
contract, dependency or decision register entry is introduced.
