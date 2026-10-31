# finite-grant-downgrade

Finite plan downgrade via production revoke, admin create and worker readback

Status: pass

Run: cce1a688-6af5-483c-bd67-2d167c51c04b

Source: 98a4b4607cef592a4b00f79aa623c15682cc6ef4

Spec: §9.2, AT02, D-050, D-081

- native_cohort: {"streams":[{"stream":"fvpn-e2e-member-finite-u01-p1/protected-device-flow","result":"scanned","scan":{"present":true,"bytes":103,"mode":"0:600","device_flow_keys":1},"deleted":true},{"stream":"fvpn-e2e-member-finite-u01-p2/protected-device-flow","result":"scanned","scan":{"present":true,"bytes":103,"mode":"0:600","device_flow_keys":1},"deleted":true},{"stream":"fvpn-e2e-member-finite-u01-p3/protected-device-flow","result":"scanned","scan":{"present":true,"bytes":103,"mode":"0:600","device_flow_keys":1},"deleted":true}],"Users":1,"Peers":3,"IdentityMatches":3,"Authorized":3,"ScannedLogs":3,"ProtectedLogs":3,"SimultaneousAuthorized":3}
- downgrade: {"revoke_replay_verified":true,"create_replay_verified":true,"member_id":"01a192cb-3839-7d3a-815a-0bb73ed16f5b","plan_revision_id":"01a192cb-8e82-72fd-8448-920218238f80","revoked_grants":["01a192cb-3842-762c-b5f1-4f04081955fa"],"operation_id":"01a192cb-8eac-7aaa-9f62-7e64ff644fcd","accepted":true,"projected":true,"before_revision":"1","after_revision":"3","device_limit":2,"authorized":2,"pending":1,"valid_until":"2026-10-31T17:29:36.271463755Z","observed_at":"2026-10-31T16:29:39.320204271Z"}
- expired_input_rejected: true
- restored: {"revoke_replay_verified":true,"create_replay_verified":true,"member_id":"01a192cb-3839-7d3a-815a-0bb73ed16f5b","plan_revision_id":"01a192c7-20bd-7514-aaa6-a23ab4c05199","revoked_grants":["01a192cb-8ea8-7c99-a9ab-23cc371cae72"],"operation_id":"01a192cb-9a93-74ca-a427-d675612917b0","accepted":true,"projected":true,"before_revision":"3","after_revision":"5","device_limit":3,"authorized":3,"pending":0,"valid_until":"2026-10-31T17:29:39.320291037Z","observed_at":"2026-10-31T16:29:44.377736985Z"}
