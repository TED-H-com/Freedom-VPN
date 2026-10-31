# second-exit-counted

Serving second exit before namespace fault

Status: fail

Run: 2454c833-ae93-48e0-96b0-00dee7cc3117

Source: 114d31df188dfa71b43c5884588e8f8f9ea958b2

Spec: AT09, F8-01

- member_route_before_selection: 192.0.3.10 dev fvnb0 table netbird src 100.122.232.86 uid 0
    cache
- netbird_networks_before_selection: Available Networks:

  - ID: fvpn-exit-421c2a1a2d0ba381bbebed3e
    Network: 0.0.0.0/0
    Status: Selected

  - ID: fvpn-exit-4a88849e9ed494e8a41b53f9
    Network: 0.0.0.0/0
    Status: Not Selected
- exit_discovery_seconds: 0.000
- exit_selection: expected exactly one API-published exit
- owner: w3-devstack-e2e: native client exit selection; see route and NetBird network evidence
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 26336294132788
wall_unix_ns 1793442798777761634
policy_expires_mono_ns 29806060200067
baseline_ok 1
serving 1
generation 3
reason serving
