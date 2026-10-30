# member-traffic

Member device and exit traffic

Status: blocked

Run: 3a129919-af3f-4bc9-b54a-8075eae8af45

Source: 79259af1facc1faa798fbb83cc20aa4e74539d1f

Spec: §8, §19 AT12

- member_id_present: true
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- plan_id: 01a18da0-bf1a-757a-914c-1c38273068cb
- plan_revision_id: 01a18da0-bf1a-77ec-8f67-148a0732a7cd
- grant: 202
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-30T16:26:32Z
- binding_valid_until: 2026-10-30T17:26:12Z
- binding: projected
- netbird_user_bindings: 1
- member_devices: 1
- device_bindings: 1
- external_user_id: [redacted]
- projection: mapping present
- checkout: 201
- member_route_before_selection: 192.0.3.10 via 172.28.30.1 dev eth0 src 172.28.30.41 uid 0
    cache
- netbird_networks_before_selection: No networks available.
- exit_selection: expected exactly one API-published exit
- owner: w3-devstack-e2e: native client exit selection; see route and NetBird network evidence
- heartbeat: v1
boot_id 692e4cdb-7abb-41db-9d5e-5143f1b49ccc
mono_ns 732633685714492
wall_unix_ns 1793377594488893304
policy_expires_mono_ns 736211492342464
baseline_ok 1
serving 1
generation 5
reason serving
- persistent_boot_recovery_config_owner_mode: 0:600
- admin_node: 200
- control_status: online
- safety_status: passed
- data_health: healthy
- provisioning_state: ready
- lifecycle_state: registered
- publication_intent: published
- publication_observed: withdrawn
- config_desired: 5
- config_applied: 3
- check_dns_policy: passed
- check_internet_only: passed
- check_lease: not_run
- check_meter_path: passed
- check_policy_signature: passed
- check_qos: not_run
- check_release_supported: passed
- check_source_binding: not_run
- check_storage: passed
- check_tunnel_probe: passed
- stored_lease_matches_applied_and_desired_generation: true
- serving_gate_rows: {"capacity": {"up_bps": 200000000, "down_bps": 200000000, "member_up_bps_cap": 100000000, "member_down_bps_cap": 100000000}, "lease_count": 5, "observation": {"checks": [{"name": "dns_policy", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}, {"name": "internet_only", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}, {"name": "lease", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}, {"name": "meter_path", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}, {"name": "policy_signature", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}, {"name": "qos", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}, {"name": "release_supported", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}, {"name": "source_binding", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}, {"name": "storage", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}, {"name": "tunnel_probe", "detail": "Egress probe used no extra target.", "status": "passed", "error_code": null, "observed_at": "2026-10-30T16:26:39.519491552Z"}], "forwarding_block_reasons": [], "member_forwarding_enabled": true}, "latest_lease": {"issued_at": "2026-10-30T16:26:34.435096+00:00", "expires_at": "2026-10-31T16:26:34.435096+00:00", "generation": "5", "member_count": 1}, "applied_generation": "5", "desired_generation": "5"}
- desired_member_rate_policies: 1
- desired_device_bindings: 1
- desired_generation: 5
- desired_publication_intent: published
- traffic: not claimed; no admission counter increment. Inspect route selection, tunnel captures and named rule counters.
- owner: w3-devstack-e2e: native client exit selection; see route and NetBird network evidence
