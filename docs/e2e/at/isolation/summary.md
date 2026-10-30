# Follow-up 3 — counted assembled isolation

AT10 **passes**. AT11, AT12 and AT13 stay **blocked** on designed gaps whose owners have not delivered. Twenty-two measured subchecks pass and five stay blocked. The four independent mechanism controls pass. The aggregate run exits **2** after writing every product record. A Go failure here is the records-first gate, not a failed new measurement.

Live run: `49799a38-1980-4c0f-a527-725646230eee`, source
`4d84f0a651e60ae84f168a8a33248503a50cc94a` (`source_dirty=false`; main baseline `dde4aa6`, which includes D-067 / PR #193).
Product UTC window: `2026-10-30T19:43:32Z`–`2026-10-30T19:57:57Z`.
Final policy generation 28. Boot release `01a18e56-ca96-7e8d-a199-5c681ab353f5`.
Node incarnation `01a18e56-d3ff-7956-8872-5c08664ad1fe`.
Image IDs are in each JSON record. Mechanism lab `ok` in 18.834 s; `TestIsolationAssembled` 865.48 s.

| Product AT | Status / evidence | Counted positive control in the same record | Deciding observation |
|---|---|---|---|
| AT10 | pass — [JSON](AT10.json), [Markdown](AT10.md) | A1 `01a18e58-5412-71cc-a654-bde23e6640e3`: RFC1918-10 0→2671, RFC1918-172 2671→5342, RFC1918-192 5342→7973, metadata 7973→10603, public-prefix-internal 10603→13275, deprecated-6to4 13275→15906, interface/generation 15906→18538 | `destination_positive_controls`: `with_positive=RFC1918-10,RFC1918-172,RFC1918-192,metadata,public-prefix-internal,deprecated-6to4; without_positive=` (empty). Every destination case has its positive and passed. Interface change generation 12→13: 3968 attempts in 60.013 s, zero delivered. |
| AT11 | blocked — [JSON](AT11.json), [Markdown](AT11.md) | Metadata 7973→10603 and deprecated-6to4 13275→15906 pass. The IPv6 check also has a positive, 21209→23839 | `IPv6 platform matrix` stays blocked. Owner `w3-devstack-e2e: real IPv6 and platform paths are devstack backlog`. A counted positive does not promote that check. |
| AT12 | blocked — [JSON](AT12.json), [Markdown](AT12.md) | Eight passing checks, including cross-member A1 26469→29100 and B1 0→2631. DNS 37072→39704 and unknown-source 39704→42374 also have positives. Publish-A withdraws B: B1 2631→5262 | `designated DNS exception` blocked, owner `w3-devstack-e2e: resolver on member underlay`. `unknown/spoofed member source` blocked, owner `w3-devstack-e2e`. |
| AT13 | blocked — [JSON](AT13.json), [Markdown](AT13.md) | Disable A1 45045→47675; revoke sibling A2 2631→5302; restart A1 50346→52978; finite-member expiry device `01a18e5f-e14d-76b2-a970-5c26cb9422c9` 0→2671; whole-node A1 55608→58228 | `short whole-node lease expiry` **pass** and `lease member expiry under control loss` **pass**. The AT stays blocked because `post-revoke kernel ingress attribution` is blocked (owner `netlab follow-up 8 → devstack follow-up 13`) and `continuous crash/NetBird/reboot/updater transitions` is blocked (owner `acceptance / operator`). |

## Convergence, publication to the first counted byte

Sampler output is [convergence-timeline.json](convergence-timeline.json): 14 samples from `2026-10-30T19:45:03.64112152Z` until the first valid counted positive. `max_offline_seconds` stayed 86400. No sample reported `STALE_GENERATION` or `policy_missing`. The only block reason is the provisional grant-less window below.

| Sample (UTC) | Desired gen / bindings | Lease gen / members | Forward chain | Observation |
|---|---|---|---|---|
| 19:45:03.641 | 5 / 0 | 5 / 0 | gen 5, accepts 0, forwarding | lease `not_run`, no block, publication `active` |
| 19:45:05.670 | 6 / 0 | 5 / 0 | baseline gen 0 | observation still gen 5; lease `not_run` |
| 19:45:07.669 | 6 / 0 | 6 / 1 | gen 6, accepts 0 | observation still gen 5 |
| 19:45:17.677 | 7 / 1 | 6 / 1 | gen 6, accepts 0 | applied 6, provisioning `applying`, publication `withdrawn` |
| 19:45:19.670 | 8 / 1 | 6 / 1 | gen 6, accepts 0 | still `applying`, `withdrawn` |
| 19:45:21.670 | 8 / 1 | 8 / 0 | baseline gen 0, forwarding false | `block_reasons=["lease_expired"]`; lease check `failed`, `LEASE_EXPIRED`, detail `Serving lease has no member grants yet.`; safety `passed`. Same generation, desired bindings nonempty, members empty: the D-067 provisional lease. |
| 19:45:27.671 | 8 / 1 | 8 / 1 | gen 8, accepts 6, forwarding | lease check `passed`, no block. About 6 s later (lab heartbeat is 5 s). Publication still `withdrawn`. |
| 19:45:31.670 | 9 / 2 | 8 / 1 | gen 8, accepts 6 | publication `active` |
| 19:45:33.670 | 10 / 2 | 9 / 1 | gen 9, accepts 12 | applied still 8 |
| 19:45:37.670 | 10 / 2 | 10 / 2 | gen 10, accepts 12 | |
| 19:45:45.670 | 11 / 3 | 10 / 2 | gen 10, accepts 12 | |
| 19:45:48 | first counted byte | | | device `01a18e58-5412-71cc-a654-bde23e6640e3`, bytes 0→2671 |
| 19:45:49.670 | 11 / 3 | 11 / 2 | gen 11, accepts 18 | |
| 19:45:57.597 and 19:45:57.629 | 11 / 3 | 11 / 2 | gen 11, accepts 18 | sampler stopped after that first byte |

The grant-less generation-8 lease was serving with one member grant before any AT10 destination probe. Every AT10 destination case therefore has its positive control on this run. Follow-up 2's generation-8 `STALE_GENERATION` snapshot and the later recovery snapshot stay in [history/follow-up-2](history/follow-up-2/); this run did not rewrite those two files, so they are not current evidence.

## AT13 expiry

Finite-member expiry ran first, while the fleet ceiling was still 86400. Signed member deadline `2026-10-30T19:56:23Z` (node lease `issued_at` `2026-10-30T19:53:34.343009Z`, node `expires_at` `2026-10-31T19:53:34.343009Z`, generation 25). Control and worker stopped at `2026-10-30T19:55:58Z`. Five later TCP requests each appeared once on `fvmb0`, none on `fvup0`, each timed out with the exit route still on `fvnb0`, and admission bytes stayed 2671. The node heartbeat still reported `serving 1` because the node lease deadline is later.

Whole-node expiry then used only `PUT /api/v1/admin/fleet-config`:

| Step | Record |
|---|---|
| Set 60 | 200, `resource_version` 3, `updated_at` `2026-10-30T19:56:46.780019Z`. Request `01a18e62-df38-76b9-a5ba-f7e336408e37`. Audit `01a18e62-df3a-7317-889d-d8985d5bc114` at `2026-10-30T19:56:46.77815Z`, action `fleet_config.update`, before 86400, after 60, reason `TEST ONLY: AT13 whole-node expiry (isolation follow-up 3)`. |
| Renewal | Capacity PATCH 202. Control and worker stopped at `2026-10-30T19:56:53Z`. |
| Node-store lease | `issued_at` `2026-10-30T19:56:52.597989849Z`, `expires_at` `2026-10-30T19:57:52.597989849Z`, generation 28, `max_offline_seconds` 60, `member_count` 2, `source=node-store`. |
| Forward after stop | Route `dev fvnb0`, wget ok, bytes 55608→58228. |
| Near deadline | Route `dev fvnb0`, wget ok, bytes 58228→60808. |
| Signed deadline | `2026-10-30T19:57:52.597989849Z`. Agent tick 5 s. |
| Baseline | `2026-10-30T19:57:53.873632379Z` (about 1.28 s after the deadline, inside one tick). `early_baseline_at` is zero. Heartbeat `baseline_ok 1`, `serving 0`, generation 28, reason `lease_expired`. |
| Drop | One packet on `fvmb0` at 19:57:55.079983, zero on `fvup0`, timeout, route still `fvnb0`, bytes frozen 60808→60808. |
| Restore 86400 | 200, `resource_version` 4, `updated_at` `2026-10-30T19:57:57.628257Z`. Request `01a18e63-f3f9-78ea-ac52-1f56de92b85e`. Audit `01a18e63-f3fb-7207-a0dc-10bcae923bae` at `2026-10-30T19:57:57.627108Z`, before 60, after 86400, reason `TEST ONLY: restore AT13 whole-node expiry ceiling (isolation follow-up 3)`. |

Both audits are `result=succeeded`, `operation_id` null, resource `fleet_config` `00000000-0000-0000-0000-000000000001`. The check passed, so restore and both audits succeeded. The live AT13 record repeats evaluator `SHA256(concatenated capture.go, capture_review_test.go)=d4cd4dfd0957cace3ef9cd63168a4653c75ac1d0ba07d5c5ca1af00603f38d53`. The singular/plural parser from follow-up 2 stayed in force and those two files were not edited.

| Mechanism control | Status | Assembled result it supports |
|---|---|---|
| [lab-AT10](lab-AT10.json) | pass | AT10 destination and interface/apply isolation; 20 regenerations, 86.7 attempts/s, positive origin retained |
| [lab-AT11](lab-AT11.json) | pass | AT11 IPv6 block and special-address mechanisms; does not prove real member-platform paths |
| [lab-AT12](lab-AT12.json) | pass | AT12 peer/source/DNS isolation; directional captures retain the outbound-capture artefact distinction from #69 |
| [lab-AT13](lab-AT13.json) | pass | AT13 conntrack revocation and baseline/crash mechanisms; does not replace assembled lifecycle or ingress proof |

See [requests](requests.md) for owners and [RUN.md](RUN.md) for reproduction.
Follow-up 2 records are under [history/follow-up-2](history/follow-up-2/).
The #69 run is under [history/follow-up-1](history/follow-up-1/follow-up-1.md).
No enforcement bypass is claimed.
