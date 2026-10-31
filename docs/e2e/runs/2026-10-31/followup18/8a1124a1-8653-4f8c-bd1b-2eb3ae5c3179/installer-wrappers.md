# installer-wrappers

ENR-06 fixed runtime layout, sampled hashes and restoration

Status: pass

Run: 8a1124a1-8653-4f8c-bd1b-2eb3ae5c3179

Source: a0089b8386c749100d9bd19b8002dd4d666d065c

Spec: §6.3, ENR-06, AT07

- registered_release_0: {"architectures":["amd64","arm64"],"manifest_sha256":"ef6a6e70ee0a5870ef81b1f21b6358edf48bf3134d5ed95f8b84afa67c77584c","release_id":"01a19192-ad63-7d61-b846-929be061b789"}
- registered_release_1: {"architectures":["amd64","arm64"],"manifest_sha256":"9802e0949bfacc17220b617e93083b6f92da3c1b19b1612e7f3782c141e6f9a0","release_id":"01a19192-b8f2-7424-864b-9b781677ba18"}
- initial: {"Exit":0,"Status":"installed","Code":"baseline","Incarnation":"01a19192-d10b-7c18-b918-36b9b6fb6a04","NodeID":"01a19192-cabd-7a26-8cdc-3049aeda33fa","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"","Before":{"key_sha256":"absent","cert_sha256":"absent","node_json_sha256":"absent","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"absent"},"Hashes":{"key_sha256":"f6928dc6476e1646f20090580acc9e4256071e794e1275c91a77f9ecb9eb6e7c","cert_sha256":"3e338a0721f4a6c03ae98aa18e761946cfe2d978757c0fba46d14115f09d2c8a","node_json_sha256":"249ee2d4b60c59191244473e4e0669ab005ad0f89a715e4a8732446295700dd8","usage_db_sha256":"b2f4507fa620b31647ed8deb37277cb312f654a7671aa1a24ad436ee5308f46b","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19192-ad63-7d61-b846-929be061b789"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- first_poll_barrier_no_desired: true
- fresh_updater_no_work: true
- rerun: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a19192-d10b-7c18-b918-36b9b6fb6a04","NodeID":"01a19192-cabd-7a26-8cdc-3049aeda33fa","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"f6928dc6476e1646f20090580acc9e4256071e794e1275c91a77f9ecb9eb6e7c","cert_sha256":"3e338a0721f4a6c03ae98aa18e761946cfe2d978757c0fba46d14115f09d2c8a","node_json_sha256":"249ee2d4b60c59191244473e4e0669ab005ad0f89a715e4a8732446295700dd8","usage_db_sha256":"68df657753635bd76dd10eacd04435b7d0c92c021d52475337291e63320f9703","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19192-ad63-7d61-b846-929be061b789"},"Hashes":{"key_sha256":"f6928dc6476e1646f20090580acc9e4256071e794e1275c91a77f9ecb9eb6e7c","cert_sha256":"3e338a0721f4a6c03ae98aa18e761946cfe2d978757c0fba46d14115f09d2c8a","node_json_sha256":"249ee2d4b60c59191244473e4e0669ab005ad0f89a715e4a8732446295700dd8","usage_db_sha256":"68df657753635bd76dd10eacd04435b7d0c92c021d52475337291e63320f9703","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19192-ad63-7d61-b846-929be061b789"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- rerun_hashes_stable: true
- netbird_owned_socket: true
- shared_runtime_survives_consumer_stop: true
- restored_services: {"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}
