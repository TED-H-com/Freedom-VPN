# finite-grant-downgrade

Finite plan downgrade via production revoke, admin create and worker readback

Status: pass

Run: a81785fa-77eb-428a-b846-fb33196a2dcf

Source: ec84d71f45ce3622b11a8a7e55f67c9c1c432d29

Spec: §9.2, AT02, D-050, D-081

- native_cohort: {"streams":[{"stream":"fvpn-e2e-member-finite-u01-p1/protected-device-flow","result":"scanned","scan":{"present":true,"bytes":103,"mode":"0:600","device_flow_keys":1},"deleted":true},{"stream":"fvpn-e2e-member-finite-u01-p2/protected-device-flow","result":"scanned","scan":{"present":true,"bytes":103,"mode":"0:600","device_flow_keys":1},"deleted":true},{"stream":"fvpn-e2e-member-finite-u01-p3/protected-device-flow","result":"scanned","scan":{"present":true,"bytes":103,"mode":"0:600","device_flow_keys":1},"deleted":true}],"Users":1,"Peers":3,"IdentityMatches":3,"Authorized":3,"ScannedLogs":3,"ProtectedLogs":3,"SimultaneousAuthorized":3}
- downgrade: {"revoke_replay_verified":true,"create_replay_verified":true,"member_id":"01a190b8-3c1b-7684-926a-11e648d8b44a","plan_revision_id":"01a190b8-7f60-738c-9832-da40928be5fb","revoked_grants":["01a190b8-3c50-7d74-ba1d-57233783992c"],"operation_id":"01a190b8-7f8c-79dd-b5ea-afc6e74c1f5f","accepted":true,"projected":true,"before_revision":"1","after_revision":"3","device_limit":2,"authorized":2,"pending":1,"valid_until":"2026-10-31T07:49:32.780600811Z","observed_at":"2026-10-31T06:49:54.923940068Z"}
- expired_input_rejected: true
- restored: {"revoke_replay_verified":true,"create_replay_verified":true,"member_id":"01a190b8-3c1b-7684-926a-11e648d8b44a","plan_revision_id":"01a190b5-8e99-71c5-87c3-2500c0a2451c","revoked_grants":["01a190b8-7f88-7566-bf97-df31fa919283"],"operation_id":"01a190b8-d609-759f-b018-2d970e517622","accepted":true,"projected":true,"before_revision":"3","after_revision":"5","device_limit":3,"authorized":3,"pending":0,"valid_until":"2026-10-31T07:49:54.924043265Z","observed_at":"2026-10-31T06:50:30.147032164Z"}
