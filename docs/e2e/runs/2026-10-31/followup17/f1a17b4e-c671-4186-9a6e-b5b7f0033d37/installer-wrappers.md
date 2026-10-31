# installer-wrappers

ENR-06 fixed runtime layout, sampled hashes and restoration

Status: pass

Run: f1a17b4e-c671-4186-9a6e-b5b7f0033d37

Source: 6e3667f57a55c2757644b2840d4a6763435f5009

Spec: §6.3, ENR-06, AT07

- registered_release_0: {"architectures":["amd64","arm64"],"manifest_sha256":"4e8437a1928290249702057be73d06e0e580011797f9d5edb4f27f4d7d56c4d8","release_id":"01a190fa-2cfb-7d5d-be3c-e01dc9e92c4d"}
- registered_release_1: {"architectures":["amd64","arm64"],"manifest_sha256":"9226b1680aa35007770700aacc32ed46a90427736c2ab9a664150334fea0e3d5","release_id":"01a190fa-36fb-7171-8a00-341758ae288f"}
- initial: {"Exit":0,"Status":"installed","Code":"baseline","Incarnation":"01a190fa-4a40-70ea-9508-f89640db0b8f","NodeID":"01a190fa-44c8-7854-834a-485f52b7c598","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"","Before":{"key_sha256":"absent","cert_sha256":"absent","node_json_sha256":"absent","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"absent"},"Hashes":{"key_sha256":"62982d4987588c8c0492c4858f18303f78bd1baae3118ab1803416fbcb0df736","cert_sha256":"d423a8be66a7176774fc7a782669db371f305271b41a876b852f29cb9093548f","node_json_sha256":"b831e309728761f3b868cef5d8eb1568e8912e4bc4c5f00d263f818a731548cf","usage_db_sha256":"acded3afb65e9c5f31b1a5e71523c2e6ab7e66a0076585a40f219e17cff8002a","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a190fa-2cfb-7d5d-be3c-e01dc9e92c4d"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- first_poll_barrier_no_desired: true
- fresh_updater_no_work: true
- rerun: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a190fa-4a40-70ea-9508-f89640db0b8f","NodeID":"01a190fa-44c8-7854-834a-485f52b7c598","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"62982d4987588c8c0492c4858f18303f78bd1baae3118ab1803416fbcb0df736","cert_sha256":"d423a8be66a7176774fc7a782669db371f305271b41a876b852f29cb9093548f","node_json_sha256":"b831e309728761f3b868cef5d8eb1568e8912e4bc4c5f00d263f818a731548cf","usage_db_sha256":"5d4b10a6a2bf08a9a8e1b2e1c2742e4d6b3c29c3c2129cd8ac56ef247004c53e","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a190fa-2cfb-7d5d-be3c-e01dc9e92c4d"},"Hashes":{"key_sha256":"62982d4987588c8c0492c4858f18303f78bd1baae3118ab1803416fbcb0df736","cert_sha256":"d423a8be66a7176774fc7a782669db371f305271b41a876b852f29cb9093548f","node_json_sha256":"b831e309728761f3b868cef5d8eb1568e8912e4bc4c5f00d263f818a731548cf","usage_db_sha256":"5d4b10a6a2bf08a9a8e1b2e1c2742e4d6b3c29c3c2129cd8ac56ef247004c53e","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a190fa-2cfb-7d5d-be3c-e01dc9e92c4d"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- rerun_hashes_stable: true
- netbird_owned_socket: true
- shared_runtime_survives_consumer_stop: true
- restored_services: {"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}
