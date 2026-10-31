# installer-wrappers

ENR-06 fixed runtime layout, sampled hashes and restoration

Status: pass

Run: 5e9fb189-1c21-4545-bb10-56ebc3f0ebdb

Source: 5b01df3d9d1c106c4e44b293f616e3da4d1da447

Spec: §6.3, ENR-06, AT07

- registered_release_0: {"architectures":["amd64","arm64"],"manifest_sha256":"f47b99eadbd0ffc7efebce75447774b98c2d5d2cc675cd8bc9fd1c4150433596","release_id":"01a19230-3d40-7239-9755-325e80ebdfe3"}
- registered_release_1: {"architectures":["amd64","arm64"],"manifest_sha256":"a80e9a07711dfb30fb08ac4250a36205bdd4e3076f0112ef8059cd101bcfb97e","release_id":"01a19230-4779-77c2-bb28-cce20d28acf9"}
- initial: {"Exit":0,"Status":"installed","Code":"baseline","Incarnation":"01a19230-5bea-74ed-86bd-248d1c5a4864","NodeID":"01a19230-56b1-7bf1-ae12-5e8b461aeb20","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"","Before":{"key_sha256":"absent","cert_sha256":"absent","node_json_sha256":"absent","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"absent"},"Hashes":{"key_sha256":"333a816c62022ddc65ed6cbcefaa8837853b0c219ef4d1b7be835eadc65ebbd1","cert_sha256":"dcb9fc8b4ab7c612a5257b37d0d11a65cf310516439778cc5ae5e572c874f88e","node_json_sha256":"088cd012678e12b6522edad89a1fd6ee8d3c3ea8e77b91c4d9b9337012c68f0c","usage_db_sha256":"8a7c2bbff637cf5320e5fb697593aab9b82479ed9bbfd233e6b249ae020be5a6","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19230-3d40-7239-9755-325e80ebdfe3"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- first_poll_barrier_no_desired: true
- fresh_updater_no_work: true
- rerun: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a19230-5bea-74ed-86bd-248d1c5a4864","NodeID":"01a19230-56b1-7bf1-ae12-5e8b461aeb20","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"333a816c62022ddc65ed6cbcefaa8837853b0c219ef4d1b7be835eadc65ebbd1","cert_sha256":"dcb9fc8b4ab7c612a5257b37d0d11a65cf310516439778cc5ae5e572c874f88e","node_json_sha256":"088cd012678e12b6522edad89a1fd6ee8d3c3ea8e77b91c4d9b9337012c68f0c","usage_db_sha256":"e37cfb14148b450f99248a9d9a40509fa90e26926c41c4ed11a7780a8523b93d","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19230-3d40-7239-9755-325e80ebdfe3"},"Hashes":{"key_sha256":"333a816c62022ddc65ed6cbcefaa8837853b0c219ef4d1b7be835eadc65ebbd1","cert_sha256":"dcb9fc8b4ab7c612a5257b37d0d11a65cf310516439778cc5ae5e572c874f88e","node_json_sha256":"088cd012678e12b6522edad89a1fd6ee8d3c3ea8e77b91c4d9b9337012c68f0c","usage_db_sha256":"e37cfb14148b450f99248a9d9a40509fa90e26926c41c4ed11a7780a8523b93d","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19230-3d40-7239-9755-325e80ebdfe3"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- rerun_hashes_stable: true
- netbird_owned_socket: true
- shared_runtime_survives_consumer_stop: true
- restored_services: {"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}
