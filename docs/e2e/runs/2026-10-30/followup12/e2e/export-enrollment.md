# export-enrollment

Harness key replay, race and actual one-time peer bootstrap

Status: pass

Run: 8b97ae7f-ea01-4fba-9a34-930e330684ea

Source: a36bb84f1dafc7526c021177026b4d558e273cd0

Spec: §6.2, AT06, ENR-04

- race_result: {"enroll_http":201,"bootstrap_http":0,"request_sha256":"420d61f5d7eded88dc55922364f358a4b6e8a214ea1d533fa185c45685a4722f","response_discarded":true,"joined":false,"node_id":"00000000-0000-0000-0000-000000000000","incarnation_id":"00000000-0000-0000-0000-000000000000"}
- race_result: {"enroll_http":409,"bootstrap_http":0,"error_code":"ENROLLMENT_KEY_MISMATCH","request_sha256":"4545b0b306d39e9428b722c2f6db3f42ca1a82e9068dd87e15ca59b7ebe420b6","response_discarded":true,"joined":false,"node_id":"00000000-0000-0000-0000-000000000000","incarnation_id":"00000000-0000-0000-0000-000000000000"}
- replay_join: {"enroll_http":200,"bootstrap_http":200,"request_sha256":"420d61f5d7eded88dc55922364f358a4b6e8a214ea1d533fa185c45685a4722f","response_discarded":false,"joined":true,"node_id":"01a18e42-f6dd-724f-aa65-d2256d12f10a","incarnation_id":"27e6c937-b8dd-48d7-8b5e-c581e183ab78"}
- peer_inventory: {"enrollment_id":"01a18e42-f6dd-724e-8d22-1a8f8a6a8e26","operation_id":"01a18e42-fbbd-7c65-8c47-2391487049f1","node_id":"01a18e42-f6dd-724f-aa65-d2256d12f10a","incarnation_id":"27e6c937-b8dd-48d7-8b5e-c581e183ab78","enrollment_state":"consumed","setup_keys":1,"setup_key_uses":1,"correlation_groups":0,"peers":1,"projected_peer_present":true,"complete":true}
- incarnation_history: current API incarnation observed; complete history/count remains ENR-03, nodes/control + PKI
