# export-enrollment

Harness key replay, race and actual one-time peer bootstrap

Status: pass

Run: 630b9f07-62d2-4db1-9737-9060732db1f1

Source: 3c25ce9daa081874988c9b09bc8c26e66d3c1b46

Spec: §6.2, AT06, ENR-04

- race_result: {"enroll_http":201,"bootstrap_http":0,"request_sha256":"9e8f2115be1d29779e72dbf7b881185ee9c15242755ca96e718fd548290214f3","response_discarded":true,"joined":false,"node_id":"00000000-0000-0000-0000-000000000000","incarnation_id":"00000000-0000-0000-0000-000000000000"}
- race_result: {"enroll_http":409,"bootstrap_http":0,"error_code":"ENROLLMENT_KEY_MISMATCH","request_sha256":"670a3ca84b9ff5e134aa83bfc5c546835b7d887d99f73314712d449d4d066f14","response_discarded":true,"joined":false,"node_id":"00000000-0000-0000-0000-000000000000","incarnation_id":"00000000-0000-0000-0000-000000000000"}
- replay_join: {"enroll_http":200,"bootstrap_http":200,"request_sha256":"9e8f2115be1d29779e72dbf7b881185ee9c15242755ca96e718fd548290214f3","response_discarded":false,"joined":true,"node_id":"01a19020-a04f-7ccf-bd47-ba24979e4284","incarnation_id":"08deb6d1-46e1-4dfa-b630-1f1fb79daa58"}
- peer_inventory: {"enrollment_id":"01a19020-a04f-7cce-acd6-97bd0e72e4f2","operation_id":"01a19020-a51d-7cb8-9c32-936867d6b0b5","node_id":"01a19020-a04f-7ccf-bd47-ba24979e4284","incarnation_id":"08deb6d1-46e1-4dfa-b630-1f1fb79daa58","enrollment_state":"consumed","setup_keys":1,"setup_key_uses":1,"correlation_groups":0,"peers":1,"projected_peer_present":true,"complete":true}
- incarnation_history: current API incarnation observed; complete history/count remains ENR-03, nodes/control + PKI
