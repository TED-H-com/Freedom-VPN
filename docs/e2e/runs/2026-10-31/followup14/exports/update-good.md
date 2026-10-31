# update-good

Good release reaches a terminal updater state

Status: pass

Run: 0a687f3a-2351-4234-b43d-882109267967

Source: 7029dbe631b8344ccd0db3991fac81e773a26ed5

Spec: §13, §19 AT23, §19 AT25

- batch: 202
- fleet_generation: 2
- target_count: 1
- desired_contains_release: true
- probe_not_member: true
- probe_route_selected: true
- probe_config_scope: one dedicated setup-key peer /32; one isolated origin /32 UDP 4443; no member device or grant
- running_candidate_executable: true
- probe_agent_config: /etc/vpn-node/agent-run.json
- updater_log_readable: true
- probe_omission_count: 0
- invalid_probe_peer: false
- exit_code: 0
- locally_succeeded: true
- rolled_back: false
- HASH_MISMATCH: false
- recovery_required: false
- ISOLATION_BASELINE_MISSING: false
- artifact url address: false
- health_json: {"dns_policy":"passed","identity_and_state":"passed","internet_only":"passed","processes":"passed","tunnel_probe":"passed"}
- health_attestation: {"agent_dns_policy":"passed","agent_internet_only":"passed","agent_tunnel_probe":"passed","agent_version":"1.0.1","dns_policy_reason":"policy in force","internet_only_reason":"baseline held","kind":"health_attest","netbird_version":"0.82.0","phase":"candidate","published":"published","tunnel_probe_reason":"overlay_up"}
- error_code: -
- result_last_error: -
- journal_excerpt: stage=locally_succeeded error_code= error_stage= error_detail= | health_json {"dns_policy":"passed","identity_and_state":"passed","internet_only":"passed","processes":"passed","tunnel_probe":"passed"} | event seq=8 stage=checking detail={"kind":"health_attest","phase":"candidate","statuses":{"dns_policy":"passed","identity_and_state":"passed","internet_only":"passed","processes":" | event seq=9 stage=locally_succeeded detail=
- probe_traffic: {"attempts":1955,"authenticated_udp_replies":20}
- probe_window: {"samples":1754,"grant_samples":4,"source_elements":1,"destination_elements":1,"max_timeout_seconds":40,"probe_up_bytes":1540,"probe_down_bytes":1540,"member_bytes_before":2631,"member_bytes_after":2631,"elements_after":0}
- probe_origin_receipt: {"probe_id":"451c51cf-147d-4196-8a7d-41573132c68d","requests":20,"source_address":"192.0.3.20"}
- probe_traversal_verified: true
- configured_probe_issued: true
- note: locally_succeeded was observed
