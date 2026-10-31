# installer-wrappers

ENR-06 fixed runtime layout, sampled hashes and restoration

Status: pass

Run: 4e59a2c4-f510-4e63-a6b9-ed4537e5fe35

Source: c75ad1957773af8f9b9e71036be357eeb491f943

Spec: §6.3, ENR-06, AT07

- registered_release_0: {"architectures":["amd64","arm64"],"manifest_sha256":"a8346593b5d5af27f887c039cb36fd92434c73cd9cd031847ce6948ad318847c","release_id":"01a1923c-3fd5-7b03-b70d-89dd97f9a6d0"}
- registered_release_1: {"architectures":["amd64","arm64"],"manifest_sha256":"9a25d89ad6db186505c59b809d76b5ddef359e41e6dfd6564b56c473f02d7b6b","release_id":"01a1923c-4a5e-7d38-94e8-4c02ec9d2aba"}
- initial: {"Exit":0,"Status":"installed","Code":"baseline","Incarnation":"01a1923c-5fb2-7e06-8754-bd2b483f6194","NodeID":"01a1923c-5a02-7279-a4f4-e24de0b588e4","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"","Before":{"key_sha256":"absent","cert_sha256":"absent","node_json_sha256":"absent","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"absent"},"Hashes":{"key_sha256":"36f5c4df2bc7d8c919a086725da56111641f689ec9f5f1c9fb00a64b78b82d4c","cert_sha256":"56ed4a75b7da5b02e3c0429f20dcd2579b6e389b4ae0901f1c3ab3554dd4cea3","node_json_sha256":"9b6d248468cb50fd998c0442854af88ac89e7c12760e7fadd700c9d531cfd506","usage_db_sha256":"d145d7fe4aa869ddc1430e7402009a1e0dcb54aa58a7ce9e69e51fd897e0bb88","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1923c-3fd5-7b03-b70d-89dd97f9a6d0"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- first_poll_barrier_no_desired: true
- fresh_updater_no_work: true
- rerun: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a1923c-5fb2-7e06-8754-bd2b483f6194","NodeID":"01a1923c-5a02-7279-a4f4-e24de0b588e4","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"36f5c4df2bc7d8c919a086725da56111641f689ec9f5f1c9fb00a64b78b82d4c","cert_sha256":"56ed4a75b7da5b02e3c0429f20dcd2579b6e389b4ae0901f1c3ab3554dd4cea3","node_json_sha256":"9b6d248468cb50fd998c0442854af88ac89e7c12760e7fadd700c9d531cfd506","usage_db_sha256":"21f522949d23b5f3743ea959a94f60283b2ca45bf97484a6ad77da5a5519dfe5","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1923c-3fd5-7b03-b70d-89dd97f9a6d0"},"Hashes":{"key_sha256":"36f5c4df2bc7d8c919a086725da56111641f689ec9f5f1c9fb00a64b78b82d4c","cert_sha256":"56ed4a75b7da5b02e3c0429f20dcd2579b6e389b4ae0901f1c3ab3554dd4cea3","node_json_sha256":"9b6d248468cb50fd998c0442854af88ac89e7c12760e7fadd700c9d531cfd506","usage_db_sha256":"21f522949d23b5f3743ea959a94f60283b2ca45bf97484a6ad77da5a5519dfe5","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1923c-3fd5-7b03-b70d-89dd97f9a6d0"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- rerun_hashes_stable: true
- netbird_owned_socket: true
- shared_runtime_survives_consumer_stop: true
- restored_services: {"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}
