# installer-wrappers

ENR-06 fixed runtime layout, sampled hashes and restoration

Status: pass

Run: d6aaa798-6c39-49f1-8fa9-5f84f696ac91

Source: a9ad1c471720a2df27f7d4aaa3a0ae1a565fa2cd

Spec: §6.3, ENR-06, AT07

- registered_release_0: {"architectures":["amd64","arm64"],"manifest_sha256":"c34a245bc536f401e4e0717929f801903a8d41b7a6118d3d450b5d9b6cadca05","release_id":"01a19246-7581-7226-bc63-2d51bcab5202"}
- registered_release_1: {"architectures":["amd64","arm64"],"manifest_sha256":"a37030bb7c9986724efc836e13966fe197ff6dc82d082cf1b98ed8678f821206","release_id":"01a19246-805d-71cc-9dc5-86452d407914"}
- initial: {"Exit":0,"Status":"installed","Code":"baseline","Incarnation":"01a19246-cb9e-70d2-9e31-e611a924c6a6","NodeID":"01a19246-b349-7974-9962-fee669622101","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"","Before":{"key_sha256":"absent","cert_sha256":"absent","node_json_sha256":"absent","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"absent"},"Hashes":{"key_sha256":"7118ad330d921e3bc30b01927a6168eb9c26668fd1cc7f8220f4e2ca68c5aa3a","cert_sha256":"b7e1187bf0c0ce1f55bf02b11ba79ee861f527df1d71c8af4de470eb780ee436","node_json_sha256":"85d982754f5c1b3d5b59d51dac606b24a2771b981d3cc282e00043ccbea91165","usage_db_sha256":"e193dcf13518a51f165f730d174f46d8cdbdb8299fdaac451a35da3786283f1a","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19246-7581-7226-bc63-2d51bcab5202"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- first_poll_barrier_no_desired: true
- fresh_updater_no_work: true
- rerun: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a19246-cb9e-70d2-9e31-e611a924c6a6","NodeID":"01a19246-b349-7974-9962-fee669622101","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"7118ad330d921e3bc30b01927a6168eb9c26668fd1cc7f8220f4e2ca68c5aa3a","cert_sha256":"b7e1187bf0c0ce1f55bf02b11ba79ee861f527df1d71c8af4de470eb780ee436","node_json_sha256":"85d982754f5c1b3d5b59d51dac606b24a2771b981d3cc282e00043ccbea91165","usage_db_sha256":"7abce2c5a12e70f40546b804d898e0ead510dd7e43758ba7ebeb737893313bf2","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19246-7581-7226-bc63-2d51bcab5202"},"Hashes":{"key_sha256":"7118ad330d921e3bc30b01927a6168eb9c26668fd1cc7f8220f4e2ca68c5aa3a","cert_sha256":"b7e1187bf0c0ce1f55bf02b11ba79ee861f527df1d71c8af4de470eb780ee436","node_json_sha256":"85d982754f5c1b3d5b59d51dac606b24a2771b981d3cc282e00043ccbea91165","usage_db_sha256":"7abce2c5a12e70f40546b804d898e0ead510dd7e43758ba7ebeb737893313bf2","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19246-7581-7226-bc63-2d51bcab5202"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- rerun_hashes_stable: true
- netbird_owned_socket: true
- shared_runtime_survives_consumer_stop: true
- restored_services: {"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}
