# export-enrollment

Harness key replay, race and actual one-time peer bootstrap

Status: pass

Run: fa5650f3-d238-496e-b7d4-718ce705fefb

Source: a36bb84f1dafc7526c021177026b4d558e273cd0

Spec: §6.2, AT06, ENR-04

- race_result: {"enroll_http":201,"bootstrap_http":0,"request_sha256":"012ce1f8e9fb5870be5cfce5006e504267e82c385ee8b4fc41e3a1fd915a91c6","response_discarded":true,"joined":false,"node_id":"00000000-0000-0000-0000-000000000000","incarnation_id":"00000000-0000-0000-0000-000000000000"}
- race_result: {"enroll_http":409,"bootstrap_http":0,"error_code":"ENROLLMENT_KEY_MISMATCH","request_sha256":"115abc3331f850eb8b090b0e9beb2e7a0c4b2344219af1eb56f298079b9e7c4f","response_discarded":true,"joined":false,"node_id":"00000000-0000-0000-0000-000000000000","incarnation_id":"00000000-0000-0000-0000-000000000000"}
- replay_join: {"enroll_http":200,"bootstrap_http":200,"request_sha256":"012ce1f8e9fb5870be5cfce5006e504267e82c385ee8b4fc41e3a1fd915a91c6","response_discarded":false,"joined":true,"node_id":"01a18e48-a8d2-7649-a720-3ee571bf5ac1","incarnation_id":"3289ee09-d29e-44ff-b8fd-92c2df87d638"}
- peer_inventory: {"enrollment_id":"01a18e48-a8d2-7648-b47a-1d1a1f920639","operation_id":"01a18e48-adb0-7638-85c0-3cb5b3baa3ba","node_id":"01a18e48-a8d2-7649-a720-3ee571bf5ac1","incarnation_id":"3289ee09-d29e-44ff-b8fd-92c2df87d638","enrollment_state":"consumed","setup_keys":1,"setup_key_uses":1,"correlation_groups":0,"peers":1,"projected_peer_present":true,"complete":true}
- incarnation_history: current API incarnation observed; complete history/count remains ENR-03, nodes/control + PKI
