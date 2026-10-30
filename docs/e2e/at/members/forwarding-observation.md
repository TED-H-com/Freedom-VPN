# Follow-up 5 forwarding observation

Run `16aa2d1b-d46f-4e88-a509-a8a9c9076006`; UTC timestamps, source/test/probe hashes, image digests, policy
revisions and both incarnations (`01a18e3f-f9e4-7683-af90-2d6d81fe910e` on `node`
and `01a18e46-3df2-7053-a540-8422dca18746` on `node2`) are in each linked JSON.
Only controlled-origin TCP/443 aggregate packet counts are retained; no payload
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
| [AT02](AT02.json) | 5 | Race loser and revoked peer: no origin receipt |
| [AT03](AT03.json) | 1 | Device admitted through actual paid reconciliation |
| [AT04](AT04.json) | 6 | Six new-connection denials after refund/expiry/suspension; persistent timing in summary |
| [AT08](AT08.json) | 2 | Four unauthorized/disabled denials, including after restart |
| [AT09](AT09.json) | 4 | Pre-stop on both exits, post-restart recovered `node2` on generation `25`, and post-recovery primary exit; one offline denial during outage |

AT02's transaction outcomes are one winner/one quota error and 3→2→3 slots.
After DELETE the former loser and a sibling carry counted traffic, revoked
source conntrack is empty, accepted health agrees, and the agent PID is
unchanged. No restart/state clear or D-050 repair is used. This passes §19 AT02's
stated quota/ownership/withdrawal basis. The strict post-revoke node-ingress drop
measurement is the [shared devstack limitation](../../runs/2026-10-30/followup10-run3/device-revoke.md),
not a claim made here. The updater-only probe is not used as member traffic.

## Resolved restart condition: F4-01 (PR #193 / D-067)

Owner: **w2-agent-core** (resolved in PR #193, commit `705170d`, under **D-067**).
In Follow-up 4 (`history/follow-up-4/AT09.json`), cold restart of `node2`,
`node2-adguard` and `node2-netns` caused central to issue a same-generation
lease with zero member grants while `node2` was not yet `ready`, and the agent
kept that empty lease until its 18-hour renewal lead, leaving forwarding off
for the entire 90-second window.

Under D-067 (#193), a serving lease with zero member grants while the desired
generation has member bindings is treated as provisional (`leaseAwaitingMemberGrants`):
`needRenew` renews at the next heartbeat and then with exponential backoff
(`1x`, `2x`, `4x` heartbeat, capped at 60 s) until a populated lease arrives.

In Follow-up 5 (`AT09.json`, run `16aa2d1b-d46f-4e88-a509-a8a9c9076006`):
1. **Pre-stop baseline (generation `25`, 7 grants):** `node2` (`01a18e46-3df2-7053-a540-8422dca18746`) holds lease `issued_at = 2026-10-30T19:36:11.464981+00:00` (`generation: "25"`, `member_grants: 7`), `desired_generation = "25"`, `applied_generation = "25"`, `member_forwarding_observed = true`. Pre-stop transfer on `node2` increments `device_upload_bytes` `0 -> 2305` and `device_download_bytes` `0 -> 2411` (`first_counted_byte_at = 2026-10-30T19:36:31.518588973Z`).
2. **Outage and cold restart (`19:36:50.392336111Z`):** `node2`, `node2-adguard` and `node2-netns` are stopped (`stopped_at = 2026-10-30T19:36:33.597391519Z`); the selected member transfer has 0 origin receipts. All three containers are started at `resumed_at = 2026-10-30T19:36:50.392336111Z` with no policy change or admin action (`desired_generation` remains `"25"`).
3. **Provisional grant-less leases and D-067 renewal sequence on generation `25`:**
   - `2026-10-30T19:36:52.908587+00:00`: `generation: "25"`, `member_grants: 0` (initial post-start lease while `node2` is not yet `ready`).
   - `2026-10-30T19:36:53.707107Z`: `node_observations` and `GET /api/v1/admin/nodes/{node2}` (`provisional_admin_node`) record `forwarding: false`, `reasons: ["lease_expired", "safety_check_failed"]`, `lease_check: {"status": "failed", "error_code": "LEASE_EXPIRED", "detail": "Serving lease has no member grants yet."}`, `safety_status: "failed"` (as noted in PR #193, the wire reason remains `lease_expired` until contract revision 10 introduces `awaiting_member_grants`).
   - `2026-10-30T19:36:58.749086+00:00`: `generation: "25"`, `member_grants: 0` (provisional renewal #1 at ~5.8 s / 1x heartbeat).
   - `2026-10-30T19:37:08.853644+00:00`: `generation: "25"`, `member_grants: 0` (provisional renewal #2 at ~10.1 s / 2x backoff).
   - `2026-10-30T19:37:24.045812Z`: `node_observations` records `safety_status: "passed"`, `forwarding: false`, `reasons: ["lease_expired"]`, `lease_check: {"status": "failed", "error_code": "LEASE_EXPIRED", "detail": "Serving lease has no member grants yet."}`, transitioning `node2` to `ready`.
   - `2026-10-30T19:37:29.064693+00:00`: `generation: "25"`, `member_grants: 7` (provisional renewal #3 at ~20.2 s / 4x backoff receives all 7 member grants on unchanged generation `25`).
   - `2026-10-30T19:37:29.131033Z`: `node_observations` and `GET /api/v1/admin/nodes/{node2}` (`recovered_admin_node`) record `forwarding: true`, `reasons: []`, `lease_check: {"status": "passed", "error_code": null, "detail": null}`, `safety_status: "passed"`, `desired_generation: "25"`, `applied_generation: "25"`, `provisioning_state: "ready"`, `publication_intent: "published"`, `publication_observed: "active"`, `member_forwarding_observed: true`.
4. **Restored member TCP forwarding (`39.236 s` to serving-ready, `43.982 s` to first counted byte):**
   - `serving_ready_at = 2026-10-30T19:37:29.628457149Z` (`39.236 s` after container start; 18 kernel `member_accept_rules` installed on generation `25`).
   - `first_counted_byte_at = 2026-10-30T19:37:34.374304456Z` (`43.982 s` after container start): member TCP attempts through `fvnb0` (`10 to_origin / 7 from_origin`), `node2` `fvmb0` (`10 to_origin / 7 from_origin`), `fvup0` (`10 to_origin / 7 from_origin`), and `origin eth0` (`10 to_origin / 7 from_origin`) succeed with `http_status: 200`, `origin_receipt: true`, `origin_snat_matches: true`, and fresh kernel counter increments (`device_upload_bytes: "0" -> "2305"`, `device_download_bytes: "0" -> "2411"`).
