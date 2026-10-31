# export-enrollment

Harness key replay, race and actual one-time peer bootstrap

Status: pass

Run: 926791b9-36ef-4051-9d44-cd7eafdc8792

Source: 9e493e8f6407dd7f091963fb7e133a11123305c3

Spec: §6.2, AT06, ENR-04

- race_result: {"enroll_http":409,"bootstrap_http":0,"error_code":"ENROLLMENT_KEY_MISMATCH","request_sha256":"812814c3fd585b7e8d8fc1d96c50476d7bacf2dd24c920ef6dc64d7f92c08936","response_discarded":true,"joined":false,"node_id":"00000000-0000-0000-0000-000000000000","incarnation_id":"00000000-0000-0000-0000-000000000000"}
- race_result: {"enroll_http":201,"bootstrap_http":0,"request_sha256":"1608bda152351d57057eb8a5d1b987969c6ba3332086504a7e27624c5d0a9676","response_discarded":true,"joined":false,"node_id":"00000000-0000-0000-0000-000000000000","incarnation_id":"00000000-0000-0000-0000-000000000000"}
- replay_join: {"enroll_http":200,"bootstrap_http":200,"request_sha256":"1608bda152351d57057eb8a5d1b987969c6ba3332086504a7e27624c5d0a9676","response_discarded":false,"joined":true,"node_id":"01a1900f-2960-7179-a427-dfcc52d7995c","incarnation_id":"d8f82380-4908-4fc3-9a91-2dbe8b4168a3"}
- peer_inventory: {"enrollment_id":"01a1900f-2960-7178-99ee-51ecc89eb42e","operation_id":"01a1900f-2e6f-793b-b277-2303b5598da3","node_id":"01a1900f-2960-7179-a427-dfcc52d7995c","incarnation_id":"d8f82380-4908-4fc3-9a91-2dbe8b4168a3","enrollment_state":"consumed","setup_keys":1,"setup_key_uses":1,"correlation_groups":0,"peers":1,"projected_peer_present":true,"complete":true}
- incarnation_history: current API incarnation observed; complete history/count remains ENR-03, nodes/control + PKI
