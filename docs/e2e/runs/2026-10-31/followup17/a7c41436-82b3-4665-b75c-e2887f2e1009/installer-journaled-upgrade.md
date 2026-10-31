# installer-journaled-upgrade

Required journaled installer upgrade; no synthetic meter checkpoint

Status: fail

Run: a7c41436-82b3-4665-b75c-e2887f2e1009

Source: d37720e9062b3ff4528ec0213d38c10d1b6309f3

Spec: UPD05, AT26, §13.5

- stage: {"Exit":0,"Status":"staged","Code":"","Incarnation":"","NodeID":"","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"staged; updater run completed; inspect fvpn-updater status for the apply result","Before":{},"Hashes":{},"Services":null}
- meter_checkpoint_absent: true
- apply_process: {"time":"2026-10-31T08:17:31.717022205Z","level":"INFO","msg":"applying installer pending release","release_id":"01a19103-3308-7f13-b620-c3efa6095f32"}
- apply_process_success: true
- durable_result: {"generation":"1","job_id":"6ee2c940-95f6-5430-bdbe-c5314fc73ffd","source":"installer","requested_release_id":"01a19103-3308-7f13-b620-c3efa6095f32","stage":"final_sample","state":"failed","journal_stage":"rolled_back","current_release_id":"01a19103-293b-7dd5-8f43-5fb038c97d37","last_good_release_id":"","seeded_baseline":{"release_id":"01a19103-293b-7dd5-8f43-5fb038c97d37","manifest_sha256":"1f6b9ac7e4abf6149f676f85470c8bc092cfd984b1a716f3906116f3dcd5a475","tree_hash":"ca52b0fb31bd24d80357d43455fd935cf1d1adf9a7f16d3fe67c0c662ca047ac","boot_id":"97c018d7-a925-4723-9127-83c097de1c8a","seeded_at":"2026-10-31T08:11:24.128474235Z"},"recovery_required":false,"last_result_state":"failed","last_rejection":null,"health":null,"candidate_health_json":null}
- current_unchanged: true
- owner: agent-core meter checkpoint; later service restart and candidate health remain unverified
