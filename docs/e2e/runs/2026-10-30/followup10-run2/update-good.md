# update-good

Good release reaches a terminal updater state

Status: pass

Run: 273496b3-9897-43b0-b289-a571d1d14561

Source: 2eeda9df03749b942c6f5afde6a7bc3a2ba8fce3

Spec: §13, §19 AT23, §19 AT25

- batch: 202
- fleet_generation: 2
- target_count: 3
- desired_contains_release: true
- probe_not_member: true
- probe_route_selected: true
- probe_config_scope: one dedicated setup-key peer /32; one isolated origin /32 UDP 4443; no member device or grant
- running_candidate_executable: true
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
- probe_traffic: {"attempts":1308,"authenticated_udp_replies":16}
- probe_window: {"samples":888,"grant_samples":3,"source_elements":1,"destination_elements":1,"max_timeout_seconds":44,"probe_up_bytes":1232,"probe_down_bytes":1232,"member_bytes_before":20982,"member_bytes_after":20982,"elements_after":0}
- probe_origin_receipt: {"probe_id":"e4095bf2-edde-429a-9a30-27919e1790b9","requests":16,"source_address":"192.0.3.20"}
- probe_traversal_verified: true
- note: locally_succeeded was observed
