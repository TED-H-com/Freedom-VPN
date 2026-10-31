# finite-grant-downgrade

Finite plan downgrade via production revoke, admin create and worker readback

Status: pass

Run: 0e34514e-bad1-4fe6-89fa-ff55da5fee12

Source: 59903b57c41115f02ccfb0b2bc1d56dc095b5a39

Spec: §9.2, AT02, D-050, D-081

- native_cohort: {"streams":[{"stream":"fvpn-e2e-member-finite-u01-p1/protected-device-flow","result":"scanned","scan":{"present":true,"bytes":103,"mode":"0:600","device_flow_keys":1},"deleted":true},{"stream":"fvpn-e2e-member-finite-u01-p2/protected-device-flow","result":"scanned","scan":{"present":true,"bytes":103,"mode":"0:600","device_flow_keys":1},"deleted":true},{"stream":"fvpn-e2e-member-finite-u01-p3/protected-device-flow","result":"scanned","scan":{"present":true,"bytes":103,"mode":"0:600","device_flow_keys":1},"deleted":true}],"Users":1,"Peers":3,"IdentityMatches":3,"Authorized":3,"ScannedLogs":3,"ProtectedLogs":3,"SimultaneousAuthorized":3}
- downgrade: {"revoke_replay_verified":true,"create_replay_verified":true,"member_id":"01a190a7-a214-7dde-bac6-888d3ab9d062","plan_revision_id":"01a190a7-df1b-74a6-bb12-2ae4e2e558ec","revoked_grants":["01a190a7-a225-7335-8f35-3931a2062f11"],"operation_id":"01a190a7-df74-7ba1-84d2-29dbff5bd711","accepted":true,"projected":true,"before_revision":"1","after_revision":"3","device_limit":2,"authorized":2,"pending":1,"valid_until":"2026-10-31T07:31:23.189737524Z","observed_at":"2026-10-31T06:31:47.559142031Z"}
- expired_input_rejected: true
- restored: {"revoke_replay_verified":true,"create_replay_verified":true,"member_id":"01a190a7-a214-7dde-bac6-888d3ab9d062","plan_revision_id":"01a190a5-5908-7cab-b20d-6f568f3a39ea","revoked_grants":["01a190a7-df6c-7c81-941e-e101f4e3084c"],"operation_id":"01a190a8-3e98-7b76-a9ea-ab49e69b598a","accepted":true,"projected":true,"before_revision":"3","after_revision":"5","device_limit":3,"authorized":3,"pending":0,"valid_until":"2026-10-31T07:31:47.559230183Z","observed_at":"2026-10-31T06:32:22.671313468Z"}
