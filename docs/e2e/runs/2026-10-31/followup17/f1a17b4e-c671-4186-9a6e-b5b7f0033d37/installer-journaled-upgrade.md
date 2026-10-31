# installer-journaled-upgrade

Required journaled installer upgrade; no synthetic meter checkpoint

Status: fail

Run: f1a17b4e-c671-4186-9a6e-b5b7f0033d37

Source: 6e3667f57a55c2757644b2840d4a6763435f5009

Spec: UPD05, AT26, §13.5

- stage: {"Exit":0,"Status":"staged","Code":"","Incarnation":"","NodeID":"","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"staged; updater run completed; inspect fvpn-updater status for the apply result","Before":{},"Hashes":{},"Services":null}
- meter_checkpoint_absent: true
- apply_process: {"time":"2026-10-31T08:05:07.51792948Z","level":"INFO","msg":"applying installer pending release","release_id":"01a190fa-36fb-7171-8a00-341758ae288f"}
- apply_process_success: true
- durable_result: {"generation":"1","job_id":"cec8c734-e1a7-5586-876b-e6d703b81901","source":"installer","requested_release_id":"01a190fa-36fb-7171-8a00-341758ae288f","stage":"final_sample","state":"failed","journal_stage":"rolled_back","current_release_id":"01a190fa-2cfb-7d5d-be3c-e01dc9e92c4d","last_good_release_id":"","seeded_baseline":{"release_id":"01a190fa-2cfb-7d5d-be3c-e01dc9e92c4d","manifest_sha256":"4e8437a1928290249702057be73d06e0e580011797f9d5edb4f27f4d7d56c4d8","tree_hash":"b7e6b213c61f61e18ac53ca1002d4b424d05fc4b715a8af79bd9f051a7d23bd9","boot_id":"97c018d7-a925-4723-9127-83c097de1c8a","seeded_at":"2026-10-31T08:01:35.161233493Z"},"recovery_required":false,"last_result_state":"failed","last_rejection":null,"health":null,"candidate_health_json":null}
- current_unchanged: true
- owner: agent-core meter checkpoint; later service restart and candidate health remain unverified
