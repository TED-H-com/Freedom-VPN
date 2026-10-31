# Follow-up 6 forwarding observation

Run `80db103b-622c-42b1-946d-4628daaa213d`; UTC timestamps, source/test/probe hashes, image digests, policy
revisions and both incarnations (`01a18ef8-e0eb-7940-a5c3-70512ec5c503` on `node`
and `01a18eff-b044-7031-9a7c-5025329d8cff` on `node2`) are in each linked JSON.
Only controlled-origin TCP/443 and injected `fvmb0` UDP/9 aggregate packet counts are retained; no payload
or credentials.

The normal serving path counts member traffic on both exits. The native client
waits for the advertised route and explicitly selects the `fvpn-exit-...`
network matching the intended node's real peer, preserving `skip_auto_apply=true`.
A positive requires the exact CA-verified origin response, authenticated origin
receipt from that node's SNAT address, and increases in that device's upload and
download counters.

| Case | Counted positive transfers | Other observed transfers |
|---|---:|---|
| [AT01](AT01.json) | 2 | Before/after Linux pause/unpause |
| [AT02](AT02.json) | 5 | Race loser and revoked peer: no origin receipt; `fvmb0` ingress injection (`ObserveMemberIngress`) proves revoked source hits `ctunku` (`+1` packet, `0` forwarded bytes) while admitted sibling hits `u_<hex>` (`+1` packet) and reaches `fvup0` + `origin` |
| [AT03](AT03.json) | 1 | Device admitted through actual paid reconciliation |
| [AT04](AT04.json) | 6 | Six new-connection denials after refund/expiry/suspension; persistent timing in summary |
| [AT08](AT08.json) | 2 | Four unauthorized/disabled denials, including after restart |
| [AT09](AT09.json) | 4 | Pre-stop on both exits, post-restart recovered `node2` on generation `23`, and post-recovery primary exit; one offline denial during outage |

AT02's transaction outcomes are one winner/one quota error and 3→2→3 slots.
After DELETE the former loser and a sibling carry counted traffic, revoked
source conntrack is empty, accepted health agrees, and the agent PID is
unchanged. Direct post-withdrawal `fvmb0` ingress injection via `ObserveMemberIngress` (PR #219, `tun-mirred`, link type `65534`, source port `40000`, destination `192.0.4.10:9`) proves that a frame from the revoked device `member-a2` (`01a18efa-99e8-75d7-9a6c-47f05b5c55e9`, `100.104.160.29`) increments `ctunku` from `{packets: 0, bytes: 0}` to `{packets: 1, bytes: 29}` (`+1` packet, `+29` bytes) with `0` forwarded bytes (`uplink_empty: true`, `origin_empty: true`) while its own counters (`u_01a18efa99e875d79a6c47f05b5c55e9` `{packets: 10, bytes: 2317}`, `d_01a18efa99e875d79a6c47f05b5c55e9` `{packets: 7, bytes: 2411}`) and all other device counters stay unchanged, whereas an injected frame from the admitted sibling `member` (`01a18ef9-ae93-7a1c-b934-20bed403bf2f`, `100.104.242.187`) increments `u_01a18ef9ae937a1cb93420bed403bf2f` from `{packets: 31, bytes: 6967}` to `{packets: 32, bytes: 6996}` (`+1` packet, `+29` bytes) and `d_01a18ef9ae937a1cb93420bed403bf2f` from `{packets: 22, bytes: 7283}` to `{packets: 23, bytes: 7340}` (`+1` packet, `+57` bytes on origin ICMP reply), appears on both `fvup0` (`2` lines) and `origin` (`2` lines), and leaves `ctunku` (`{packets: 1, bytes: 29}`) and all other device counters unchanged. No restart/state clear or D-050 repair is used, and the updater-only probe is not used as member traffic.

## Resolved restart condition: F4-01 (PR #193 / D-067, PR #213 & #216 revision 11)

Owner: **w2-agent-core** (resolved in PR #193 under **D-067**, with contract revision 11 wire alignment in PR #213 and PR #216).
In Follow-up 4 (`history/follow-up-4/AT09.json`), cold restart of `node2`,
`node2-adguard` and `node2-netns` caused central to issue a same-generation
lease with zero member grants while `node2` was not yet `ready`, and the agent
kept that empty lease until its 18-hour renewal lead, leaving forwarding off
for the entire 90-second window.

Under D-067 (#193) and contract revision 11 (#213, #216), a serving lease with
zero member grants while the desired generation has member bindings is issued
with a 1-minute provisional TTL and treated as provisional (`leaseAwaitingMemberGrants`):
`needRenew` renews at the next heartbeat and then with exponential backoff
(`1x`, `2x`, `4x` heartbeat, capped at 60 s) until a populated lease arrives,
while reporting `forwarding_block_reasons: ["awaiting_member_grants"]` (plus
`safety_check_failed` before safety passes), `error_code: "AWAITING_MEMBER_GRANTS"`
on `lease`, `qos`, and `source_binding` checks, and a single `stage: "lease"`
entry (`code: "AWAITING_MEMBER_GRANTS"`) in `errors[]` — never `lease_expired`
while the provisional lease is in force.

In Follow-up 6 ([AT09.json](AT09.json), run `80db103b-622c-42b1-946d-4628daaa213d`):
1. **Pre-stop baseline (generation `23`, 7 grants):** `node2` (`01a18eff-b044-7031-9a7c-5025329d8cff`) holds lease `issued_at = 2026-10-30T22:59:25.120747+00:00` (`generation: "23"`, `member_grants: 7`), `desired_generation = "23"`, `applied_generation = "23"`, `member_forwarding_observed = true`. Pre-stop transfer on `node2` increments `device_upload_bytes` `0 -> 2305` and `device_download_bytes` `0 -> 2411` (`first_counted_byte_at = 2026-10-30T22:59:44.918298551Z`).
2. **Outage and cold restart (`23:00:03.796010637Z`):** `node2`, `node2-adguard` and `node2-netns` are stopped (`stopped_at = 2026-10-30T22:59:47.018508199Z`); the selected member transfer has 0 origin receipts. All three containers are started at `resumed_at = 2026-10-30T23:00:03.796010637Z` with no policy change or admin action (`desired_generation` remains `"23"`).
3. **Provisional grant-less leases and D-067 / revision 11 renewal sequence on generation `23`:**
   - `2026-10-30T23:00:04.349901+00:00`: `generation: "23"`, `member_grants: 0` (initial post-start provisional lease with 1-minute TTL while `node2` is not yet `ready`).
   - `2026-10-30T23:00:04.375940436Z`: `node_observations` and `GET /api/v1/admin/nodes/{node2}` (`provisional_admin_node`) record `forwarding: false`, `reasons: ["awaiting_member_grants", "safety_check_failed"]` (`saw_lease_expired: false`), `lease_check: {"status": "failed", "error_code": "AWAITING_MEMBER_GRANTS", "detail": "Serving lease has no member grants yet."}`, `qos_check.error_code: "AWAITING_MEMBER_GRANTS"`, `source_binding_check.error_code: "AWAITING_MEMBER_GRANTS"`, `lease_stage_errors_count: 1`, `safety_status: "failed"`.
   - `2026-10-30T23:00:09.419186+00:00`: `generation: "23"`, `member_grants: 0` (provisional renewal #1 at ~5.1 s / 1x heartbeat).
   - `2026-10-30T23:00:19.549439+00:00`: `generation: "23"`, `member_grants: 0` (provisional renewal #2 at ~10.1 s / 2x backoff).
   - `2026-10-30T23:00:34.753993516Z`: `node_observations` records `safety_status: "passed"`, `forwarding: false`, `reasons: ["awaiting_member_grants"]` (`saw_lease_expired: false`), `lease_check: {"status": "failed", "error_code": "AWAITING_MEMBER_GRANTS", "detail": "Serving lease has no member grants yet."}`, `qos_check.error_code: "AWAITING_MEMBER_GRANTS"`, `source_binding_check.error_code: "AWAITING_MEMBER_GRANTS"`, `lease_stage_errors_count: 1`, transitioning `node2` to `ready`.
   - `2026-10-30T23:00:39.805556+00:00`: `generation: "23"`, `member_grants: 7` (provisional renewal #3 at ~20.3 s / 4x backoff receives all 7 member grants on unchanged generation `23`).
   - `2026-10-30T23:00:39.865783068Z`: `node_observations` and `GET /api/v1/admin/nodes/{node2}` (`recovered_admin_node`) record `forwarding: true`, `reasons: []`, `lease_check: {"status": "passed", "error_code": null, "detail": null}`, `safety_status: "passed"`, `desired_generation: "23"`, `applied_generation: "23"`, `provisioning_state: "ready"`, `publication_intent: "published"`, `publication_observed: "active"`, `member_forwarding_observed: true`.
4. **Restored member TCP forwarding (`36.976 s` to serving-ready, `41.695 s` to first counted byte):**
   - `serving_ready_at = 2026-10-30T23:00:40.771707223Z` (`36.976 s` after container start; 18 kernel `member_accept_rules` installed on generation `23`).
   - `first_counted_byte_at = 2026-10-30T23:00:45.491347556Z` (`41.695 s` after container start): member TCP attempts through `fvnb0` (`11 to_origin / 7 from_origin`), `node2` `fvmb0` (`11 to_origin / 7 from_origin`), `fvup0` (`11 to_origin / 7 from_origin`), and `origin eth0` (`11 to_origin / 7 from_origin`) succeed with `http_status: 200`, `origin_receipt: true`, `origin_snat_matches: true`, and fresh kernel counter increments (`device_upload_bytes: "0" -> "2305"`, `device_download_bytes: "0" -> "2461"`).
