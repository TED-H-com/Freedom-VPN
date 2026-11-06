# D164 isolation acceptance

The complete scheduled-mode Docker lab and assembled run on 2026-11-06 passed.
It closes F17-01's missing pure stale-generation evidence and supports retirement
of the D-163 exception. A future missing pure stale-generation case now fails
scheduled review.

| Recorded result | Evidence |
|---|---|
| Source | `8286ef1176065d0dfc2abf9639c2262400492721`, clean throughout |
| Run | `5c1d7234-7eb3-41a9-8d4e-1536572c4917`; one invocation, no rerun |
| Supervisor bounds | `2026-11-06T20:02:50.830477Z`–`2026-11-06T20:24:57.927245Z` (1327.096768 seconds) |
| Scheduled result | Exit 0; all 12 phases exit 0; 27 acceptance rows pass and 2 remain expected blocked |
| Populated holds | All 9 coverage items pass; 18 accepted-report/own-signed-lease joins independently checked, no join problems |
| Clone and stops | All 9 clone criteria established; all 6 parent stop receipts matched |
| Recovery and boundary | All 6 recovery assertions pass; unchanged boundary review passes |
| Run isolation and cleanup | 45 overlap samples, no foreign suite detected; owned containers, images, networks and volumes empty |

The clone collector retained an initial incoherent source read and selected a
coherent read about 491 ms later, within its existing 20-second budget. The
same-launch bracket and three-second latch predicates passed unchanged. The
three populated-hold incidents each cover a purpose-specific subset; their
aggregate coverage is complete.

The [machine-readable record](d164-acceptance.json) contains the exact coverage,
remaining limits and selected original artifact hashes and sizes. Full artifacts
are retained by the acceptance owner; this page does not provide a public artifact
bundle. Independent review checked the original same-run records and hashes.
The original scheduled output still embeds the then-active exception description;
that output and all earlier failed runs remain unchanged.

The expected blocks remain **AT11 IPv6 platform matrix** and **AT13 continuous
crash/NetBird/reboot/updater transitions**. Physical arm64, independent-clock
guests and complete real-platform/operator acceptance remain separate. Clone
results establish bounded transport and local latch evidence, not indefinite
persistence or clone datapath correctness. Stop receipt times are observation
bounds. The preflight's historical strict `r<=3` context was false for one
`r=4` sample; current admission passed, and that distinction remains recorded.
The separate scheduled E2E pre-positive failure is not resolved by this run.
