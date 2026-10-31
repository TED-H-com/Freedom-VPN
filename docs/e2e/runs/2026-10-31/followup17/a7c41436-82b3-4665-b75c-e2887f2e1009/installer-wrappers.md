# installer-wrappers

ENR-06 fixed runtime layout, sampled hashes and restoration

Status: pass

Run: a7c41436-82b3-4665-b75c-e2887f2e1009

Source: d37720e9062b3ff4528ec0213d38c10d1b6309f3

Spec: §6.3, ENR-06, AT07

- registered_release_0: {"architectures":["amd64","arm64"],"manifest_sha256":"1f6b9ac7e4abf6149f676f85470c8bc092cfd984b1a716f3906116f3dcd5a475","release_id":"01a19103-293b-7dd5-8f43-5fb038c97d37"}
- registered_release_1: {"architectures":["amd64","arm64"],"manifest_sha256":"a5f84dc51369af112002594f6ba45a241fc1df6123b1eaf85fdc335091236587","release_id":"01a19103-3308-7f13-b620-c3efa6095f32"}
- initial: {"Exit":0,"Status":"installed","Code":"baseline","Incarnation":"01a19103-46ea-7c12-882c-cb2d19a4a1b4","NodeID":"01a19103-41ed-756e-b07f-b6e58fb0724f","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"","Before":{"key_sha256":"absent","cert_sha256":"absent","node_json_sha256":"absent","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"absent"},"Hashes":{"key_sha256":"5815acc32a94ddd915143c1c5806ace172d986188f2075914c38387d972278a5","cert_sha256":"a9356dc14b68b205393135417b4eb82e089dd1e1d2e10ca455a873ce5cc58cce","node_json_sha256":"547c10e19fafbf6221388bd73416d6981b502e2631e4aec90d6b2afe006ab611","usage_db_sha256":"2c5dc17759852977dbde1ef3db8bc481970a3d7e9087f1228c5b6177a940ec56","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19103-293b-7dd5-8f43-5fb038c97d37"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- first_poll_barrier_no_desired: true
- fresh_updater_no_work: true
- rerun: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a19103-46ea-7c12-882c-cb2d19a4a1b4","NodeID":"01a19103-41ed-756e-b07f-b6e58fb0724f","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"5815acc32a94ddd915143c1c5806ace172d986188f2075914c38387d972278a5","cert_sha256":"a9356dc14b68b205393135417b4eb82e089dd1e1d2e10ca455a873ce5cc58cce","node_json_sha256":"547c10e19fafbf6221388bd73416d6981b502e2631e4aec90d6b2afe006ab611","usage_db_sha256":"0fda34319a2b04b5319b194e7822bccd4074dc3e80b75b4108a52b262da9d19a","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19103-293b-7dd5-8f43-5fb038c97d37"},"Hashes":{"key_sha256":"5815acc32a94ddd915143c1c5806ace172d986188f2075914c38387d972278a5","cert_sha256":"a9356dc14b68b205393135417b4eb82e089dd1e1d2e10ca455a873ce5cc58cce","node_json_sha256":"547c10e19fafbf6221388bd73416d6981b502e2631e4aec90d6b2afe006ab611","usage_db_sha256":"0fda34319a2b04b5319b194e7822bccd4074dc3e80b75b4108a52b262da9d19a","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19103-293b-7dd5-8f43-5fb038c97d37"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- rerun_hashes_stable: true
- netbird_owned_socket: true
- shared_runtime_survives_consumer_stop: true
- restored_services: {"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}
