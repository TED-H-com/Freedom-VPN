# rollback-delivery-finding

> Historical follow-up 1 evidence, before D-062. See [current requests](requests.md) and [summary](summary.md).

Status: blocked

Run: 86ed9a70-2786-4e36-a23d-51dad278f6e2

Source: 8adbcbde45ae61d7719847529d503a3e430068d4

Scope: Supplemental diagnostic run before case reordering; not the final make-target run

Local rollback restored 1.0.1 and its D-053 checks passed; the serialized rolled_back result retained the failed candidate tunnel health and was left pending with STORE_UNAVAILABLE.

- artifact_sha256_0.1.0_amd64: "99bb8603f2694e75753823616e1bee66e4fa8f3d6d611a71318c2b79995ae5ae"
- artifact_sha256_0.1.0_arm64: "90c03e61211d122dcae75f778f2c08e9f644ab2fd7dabc3ec54308de7d8d1a60"
- signed_manifest_sha256_0.1.0: "cc5c2f6f44a2335238a97340277081f65fa34e4428cdb69b2482b55778451076"
- manifest_sha256_0.1.0: "56522d94468269fd883119c7cf7a71511238836fec6905c30a786fa63836dc47"
- artifact_sha256_1.0.1_amd64: "6879f89481cb65f87016c6f726aa60f6a00bc4b5b03cf60f18e1995f6cdc0eb6"
- artifact_sha256_1.0.1_arm64: "bb776ce4cfc294bf23aad7c371dfe7f981b6b7fee741b65f350e2d534dc570a7"
- signed_manifest_sha256_1.0.1: "cf9fe5fa35281d7c59ac3f7d1443acf83a23a0a15b2e800e3c4dd352a582ac47"
- manifest_sha256_1.0.1: "ab7362cff6e06af93c54fa937ebdc118725461a9b3649ee9ba81b107e2302422"
- artifact_sha256_1.0.2_amd64: "bdb453f2ba490e8d47b9b9f7618b1ed0e5376a62fbd8bf43541d590bca84b94d"
- artifact_sha256_1.0.2_arm64: "04315a1ec6aa53a3551c3e5d2d6f76d9eedb0cf377b687a7c857399140bf8c70"
- signed_manifest_sha256_1.0.2: "6d3a911ec180fa4334301274e1057833aa57e1b4d569869658f799f1fafcf589"
- manifest_sha256_1.0.2: "78befbeb754e4d50a2151e216a5a8fba170a0e6c1a15e94e8c66016eab3302dd"
- verify_01a18a9d-3a67-7bbd-b5a1-6b59a138703e: "verified"
- verify_01a18a9d-423b-7353-a2f6-5135dd7e27e5: "verified"
- verify_01a18a9d-4a20-7b58-bb82-1a0cf5725129: "verified"
- accepted_unpublished_baseline: true
- netbird_manifest_version: "0.82.0"
- release_ids: {"boot":"01a18a9d-3a67-7bbd-b5a1-6b59a138703e","good":"01a18a9d-423b-7353-a2f6-5135dd7e27e5","negative":"01a18a9d-4a20-7b58-bb82-1a0cf5725129"}
- batch_response: "202"
- batch_id: "01a18a9f-6c6d-77c0-a54e-ece27220c2dc"
- generation: "2"
- signed_intent_for_this_batch: true
- failure_stimulus: "TEST ONLY: candidate TCP egress probe dials its own closed loopback port; no forced health values or journal writes"
- updater_finished: true
- journal: {"txn_id":"01a18a9f-9d62-7b72-a953-24aacc7b22b2","stage":"rolled_back","error":"HEALTH_CHECK_FAILED","job_error":"HEALTH_CHECK_FAILED","current":"01a18a9d-423b-7353-a2f6-5135dd7e27e5","last_good":"01a18a9d-423b-7353-a2f6-5135dd7e27e5","health":{"dns_policy":"passed","identity_and_state":"passed","internet_only":"passed","processes":"passed","tunnel_probe":"failed"},"attestations":[{"agent_dns_policy":"passed","agent_internet_only":"passed","agent_tunnel_probe":"failed","agent_version":"0.1.0","dns_policy_reason":"policy in force","internet_only_reason":"baseline held","kind":"health_attest","netbird_version":"0.82.0","phase":"candidate","published":"unpublished","tunnel_probe_reason":"overlay_partial"},{"agent_dns_policy":"passed","agent_internet_only":"passed","agent_tunnel_probe":"passed","agent_version":"1.0.1","dns_policy_reason":"policy in force","internet_only_reason":"baseline held","kind":"health_attest","netbird_version":"0.82.0","phase":"rollback","published":"unpublished","tunnel_probe_reason":"overlay_up"}],"outbox_state":"rolled_back","outbox_status":"pending","outbox_error":"STORE_UNAVAILABLE"}
- last_good_process_restored: true
- central: {"state":"queued","unconfirmed":true,"needs_attention":false,"current_release":"01a18a9d-423b-7353-a2f6-5135dd7e27e5","error_code":""}
- serialized_rollback_result: {"state":"rolled_back","stage":"report","error_code":"HEALTH_CHECK_FAILED","health":{"dns_policy":"passed","identity_and_state":"passed","internet_only":"passed","processes":"passed","tunnel_probe":"failed"},"requested_release_id":"01a18a9d-3a67-7bbd-b5a1-6b59a138703e","current_release_id":"01a18a9d-423b-7353-a2f6-5135dd7e27e5","started_at":"2026-10-30T02:24:38Z","finished_at":"2026-10-30T02:24:40Z","outbox_status":"pending","outbox_error":"STORE_UNAVAILABLE"}
- result_delivery_http: {"at":"2026-10-30T02:24:40.4765966Z","status":422,"source":"read-only owned control service log during the diagnostic run"}
