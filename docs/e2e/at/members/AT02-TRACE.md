# AT02 and AT04 withdrawal traces — Follow-up 13

Run `4182dd3d`, harness `d49b535a`. §19 AT02/AT04; MEM03, ENT02.
Command/read brackets and poll times are not exact kernel packet-cutoff times.

| Case / cohort | Event UTC | Last TCP / UDP reply delta (s) | Samples / unreadable | Result |
|---|---|---:|---:|---|
| AT02 / member-a4 | 2026-11-01T08:26:57.674685121Z | 0.323031048 / 0.323031048 | 29 / 0 | pass |
| AT02 / member-a4 | 2026-11-01T08:28:56Z | -0.159147674 / -0.159147674 | 208 / 0 | pass |
| AT04 / member-paid | 2026-11-01T08:33:46.134832007Z | 3.558769767 / 3.558769767 | 196 / 1 | fail |
| AT04 / member-paid2 | 2026-11-01T08:33:46.134832007Z | 3.506849756 / 3.506849756 | 250 / 0 | pass |
| AT04 / member-expiry1 | 2026-11-01T08:37:51Z | 0.120223885 / 0.120223885 | 342 / 0 | pass |
| AT04 / member-expiry2 | 2026-11-01T08:37:51Z | -0.116356178 / -0.116356178 | 346 / 0 | pass |
| AT04 / member-suspend1 | 2026-11-01T08:40:47.916475484Z | 2.454516384 / 2.454516384 | 197 / 0 | pass |
| AT04 / member-suspend2 | 2026-11-01T08:40:47.916475484Z | 2.596756019 / 2.596756019 | 203 / 1 | fail |

**AT02 under provisional D-135 + D-136:** 95 successful Applies, zero failed Applies; F11-03 stays **blocked on no failed Apply observed**. All 61 baseline calls are paired to the next successful Apply, with original rows/run identities. Three overlap the sibling window; the other 58 have command recovery evidence and explicitly unavailable sibling coverage. **4,029 attempts / 4,029 ingresses / 4,008 egresses / 21 losses**; 13 inside completed guards, eight outside. No losses are excluded. Sequences **402–404** are successful `policy_missing` baselines on the generation-11 lease, **405** renews to 13, **406** applies successfully. The first bracket is **219.395230 ms**, contains nine losses (one inside a guard, eight outside), and has first subsequent egress **14.828914 ms** after Apply. Repeated baseline brackets overlap and are never summed as unique losses. The no-loss row stays FAIL, even though recovery is measured. [Full accounting](F13-AT02-recovery.json), [original transition](F13-AT02-policy-missing.json).

**F13-01:** AT04 has two separate failed persistent-withdrawal checks: primary refund and secondary suspension each contain one `nft_parse` / `parse` sample with command exit 0. The remaining status reads succeed; observed reply cutoffs are 3.558769767 s and 2.596756019 s, with active senders and stable counters. Incomplete reads keep both rows failed. Raw nft output/parser errors were not retained, so their cause is unproven. [Failed rows, neighbors and producer windows](F13-01-counter-parse.json); shared reader/parser changes are handed off.

D-134 does not waive pre-D or renewal losses. Historical b98b78ee traces remain
[immutable](history/follow-up-12/AT02-TRACE.md).
