# D-104 — monotonic member deadline contract, 2026-10-31

The measured quantum is **1 ms**. The enforced render/install budget is
**100 ms**. With **2.9 s** initially remaining, the retained enforce-level trace
has its last positive at **D − 100.707 ms**, first dropped send at
**D − 0.837 ms**, **26 positives, 7 drops**, and no later positive. This satisfies
the requested last-positive bound of D − 101 ms and first-drop bound of D +
100 ms. This mechanism result does **not** promote assembled AT13 / ISO-F6-03;
agent-core follow-up 24 and isolation follow-up 8 must consume the contract.

[Complete one-clock trace](G2-deadline-evidence-2026-10-31/TestD104_AT13_DeadlineContinuity/trace.json)
includes packet IDs, scheduled/send/ingress/egress/read-completion times,
deadline, Apply timing, precision-read timing and tool versions. All elapsed
timestamps are decimal strings in nanoseconds on CLOCK_MONOTONIC. Captures are
userspace reads of AF_PACKET sockets, not hardware timestamps. A drop means
ingress observed and no matching egress within the retained 80 ms window.
Traffic stays inside the isolated Docker topology.

## Measured platform and precision

The live amd64 kernel is `7.0.0-38-generic`; nft is `1.0.9-1ubuntu0.1`
(`nftables v1.0.9`). The existing Ubuntu 24.04 lab image was used without a pin
change. The host kernel is shared by the container; an image alone does not
pin it. The evidence records the versions actually executed.

Eight elements use timeout 1001–1008 ms and explicit expires 701–708 ms. Text
read-back preserves every 1 ms timeout step and returns expires 699–706 ms,
also in 1 ms steps. The same kernel state through nft JSON reports timeout
`1` and expires `0` for all eight. JSON cannot establish this contract. Kernel
timeouts are jiffies, not a userspace sleep: the upstream
[Linux 7.0 implementation](https://github.com/torvalds/linux/blob/v7.0/net/netfilter/nf_tables_api.c)
converts netlink milliseconds to jiffies and initializes expiration during
element allocation. The experiment establishes the effective 1 ms quantum on
this kernel. The [nft element-timeout contract](https://wiki.nftables.org/wiki-nftables/index.php/Element_timeouts)
distinguishes original timeout from remaining expires.

Other kernel configurations, physical arm64 and suspend/resume remain
**待實測**. Do not claim that an arbitrary kernel has this quantum; run this
qualification on the supported release/platform before enabling the contract.
No cross-compilation result is a real-device measurement.

## Interface for agent-core follow-up 24

`node/enforce/model.go` exposes `DeadlineGranularity = time.Millisecond` and
`DeadlineInstallBound = 100*time.Millisecond`. `MemberLease.DeadlineMono` is
`*time.Duration`: an absolute elapsed time on Linux CLOCK_MONOTONIC, using the
same clock as `timeanchor.Mono()`. Nil preserves the legacy caller interface;
non-nil zero/negative/expired deadlines admit no element. It is never a wall
timestamp or a duration relative to process start.

In `decide.go`:

1. Keep the precise `MemberDeadline` result, bounded by the whole-node policy
   deadline and all applicable service/binding limits. Pass it through
   `MemberLease.DeadlineMono`; copy each value before taking its address.
2. Pass `Model.NowMono = in.MonoNow`, paired with the existing trusted/signed
   `Model.Now` snapshot. Apply resamples CLOCK_MONOTONIC and ages the wall
   snapshot by elapsed time only. It never refreshes allowance from the wall
   clock. Required signed window inputs and binding checks remain mandatory.
3. Remove the `rem / time.Second` admission gate and stop setting
   `TimeoutCeilSeconds` from floored seconds. That legacy field is still an
   additional cap if set, so leaving it populated preserves the old loss.
   Render admits a positive remaining quantum; Apply deducts the installation
   budget and omits an element when the resulting allowance is below 1 ms.
4. Preserve the applied model/deadlines for `VerifyServing`; do not replace its
   original sample with a newer one while retaining old original timeouts.
   The precision reader samples the current clock itself. Keep the f23 final
   heartbeat/expiry guard: a read spanning expiry cannot refresh serving.
5. Use `RenderPlan.ElementTimeouts` (`time.Duration`) for exact rendered values.
   `RenderPlan.Timeouts` is a deprecated, floored seconds projection retained
   only so the existing f23 diagnostic compiles. It never drives installation
   or verification. Replace that diagnostic with the sub-second continuity
   assertion, then retire its legacy use in a later cleanup.

Pure `Render` produces a plan for review/tests. Production installation must
use `Apply`, which performs the timing check and guarded commit. Serializing
and later replaying a rendered relative timeout is not authorized by this API.

## Installation proof and failure behavior

Let R be the final monotonic sample before rendering, D the minimum deadline,
G = 1 ms, and B = 100 ms. Apply uses

`T = floor((D − R − B) / G) * G`, omitting T < G.

Ownership reads and tc shaping precede R, so their elapsed time is removed
before rendering. The atomic nft transaction installs both the candidate and
`inet fvpn / fv_deadline_guard`, a separate forward-hook chain that drops both
directions of the member interface. Its policy accepts unrelated forwarding;
it does not change the input/output control path, routes, DNS or tc objects.

Apply samples CLOCK_MONOTONIC after the transaction returns. Only when this
upper bound on installation is within B does it delete the guard in a second
nft transaction. Thus the kernel's expiration is no later than R + B + T ≤ D.
Removing the guard never restarts the element timeout. A delayed deletion can
only reduce availability. A late commit, command cancellation, crash or pause
before the budget check leaves the candidate guarded; rollback installs the
baseline, and a failed rollback leaves the guard. This safety argument does
not assume the scheduler or context cancellation can enforce a hard latency
bound. An interrupted guard is rejected by ordinary serving verification;
fresh Apply can replace it, and baseline removes it atomically with denial.

The trace records Apply-start → installed-by **30.700 ms**, final-render-start
→ installed-by **15.345 ms**. These are measured upper bounds for this run,
not a scheduler guarantee or a claim to know the exact kernel allocation
instant. The retained member read shows `timeout 2s784ms expires 2s730ms`. `ApplyResult.Deadline` exposes all samples, G, B and guard release.
Every Apply includes a brief member-only denial between guarded commit and
release; repeated-renewal continuity must be measured in assembled acceptance.
D-090's retained link quarantine, foreign-tc ownership and recovery protocol
are unchanged. The new guard is wholly within the owned nft table.

## Read-back and D-085 budget delta

The JSON ruleset, tc attachment and policy-rule reads remain. After a valid
JSON serving listing, `VerifyServing` adds exactly **one**
`nft list set inet fvpn allowed_v4` text read. Its measured duration here is
**1.528 ms**, exposed as `ServingEvidence.DeadlineRead.Start/End`. Recovery's
existing second verification also performs that one extra read. No extra tc
attachment reads are introduced. Apply adds one guard-release nft mutation.

The strict text parser checks the table/set/type/timeout flag, unique /32
addresses and integer durations through milliseconds. It rejects unbounded,
unknown, duplicate, malformed and overflowing elements. Original timeout is
bounded by the applied model's original allowance; expires is bounded by the
remaining allowance at read start and by timeout. Equality is accepted at the
quantum boundary; no one-second tolerance, rounding up or JSON zero-as-live
inference is used. Missing elements are safe denial, not a continuity proof.

## Assertions for isolation follow-up 8

Retain f23's signed wall times, persisted monotonic send anchor and computed
policy deadline. Capture Apply start/render/install times and the new read
start/end alongside renewal, watchdog and heartbeat events. Send 100 ms probes
across D, with packet IDs and ingress/egress/read-completion on one clock.

Require last positive ≥ D − (G + B), first drop ≤ D + 100 ms, no positive at or
after D and no later positives after the first drop. The mechanism fixture
phases its cadence at D − (B + G); an arbitrary 100 ms sampling phase cannot
resolve a 1 ms boundary without an extra sample. Do not silently add a cadence
to the allowed early-expiry bound. Preserve failed acceptance until the full
agent/stack proves this continuity plus denial; passing this enforce-only
fixture is not the full continuous crash/NetBird/reboot/updater matrix.

## Reproduction

Run `make -f mk/netlab.mk netlab-lint netlab-unit netlab-integration netlab-build`.
The privileged `netlab-test` target includes `TestD104_AT13_DeadlineContinuity`
and tears its container/network down. Use `FVPN_LAB_PREFIX=fvpn-w1-netlab-d104`
on this lane. To retain just the deadline trace, bring the lab up, run the
tagged test with `-run TestD104` and `FVPN_NETLAB_EVIDENCE_DIR` pointing at the
dated evidence directory, then use `netlab-down`. Use disk-backed Go temp
directories. Check active gate runners before starting an integration suite.
