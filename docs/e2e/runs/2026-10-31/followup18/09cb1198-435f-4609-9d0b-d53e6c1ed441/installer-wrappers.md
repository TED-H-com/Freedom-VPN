# installer-wrappers

ENR-06 fixed runtime layout, sampled hashes and restoration

Status: pass

Run: 09cb1198-435f-4609-9d0b-d53e6c1ed441

Source: 84b6a342970ffc651aafda81d485451529902233

Spec: §6.3, ENR-06, AT07

- registered_release_0: {"architectures":["amd64","arm64"],"manifest_sha256":"b65a4730f6bcc0f73fabf6de0dbb0f1580a247590ba3ddbb48bb9694f6888686","release_id":"01a1917d-cdde-777b-9e4c-6caa1495d8d8"}
- registered_release_1: {"architectures":["amd64","arm64"],"manifest_sha256":"b0823f56ed891691aae6486785c986df5a38b160a85004fc77d7df0723f84dfd","release_id":"01a1917d-d8bf-7212-a6a7-79e0a7c4601e"}
- initial: {"Exit":0,"Status":"installed","Code":"baseline","Incarnation":"01a1917d-eedf-75ab-a77e-babbc7fb81dc","NodeID":"01a1917d-e899-7f0f-a666-35b7f0ddd137","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"","Before":{"key_sha256":"absent","cert_sha256":"absent","node_json_sha256":"absent","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"absent"},"Hashes":{"key_sha256":"669eaca2922a922794cc66f21498dbb09bb98c2c6d5b42ec7e983818df0e74ed","cert_sha256":"70bd5bfc06daf6b2bee486aab5ae422fef4ae68420e45105accb279d453986d4","node_json_sha256":"c0843540cd6c7cf3e79175baadbf83d724e32863cbd528581670f52d3a79f788","usage_db_sha256":"3edc0b9dc50fd2917bf563d7d032c83fe6df99c37ec7ffc1443e9bdcfd2f05fa","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1917d-cdde-777b-9e4c-6caa1495d8d8"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- first_poll_barrier_no_desired: true
- fresh_updater_no_work: true
- rerun: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a1917d-eedf-75ab-a77e-babbc7fb81dc","NodeID":"01a1917d-e899-7f0f-a666-35b7f0ddd137","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"669eaca2922a922794cc66f21498dbb09bb98c2c6d5b42ec7e983818df0e74ed","cert_sha256":"70bd5bfc06daf6b2bee486aab5ae422fef4ae68420e45105accb279d453986d4","node_json_sha256":"c0843540cd6c7cf3e79175baadbf83d724e32863cbd528581670f52d3a79f788","usage_db_sha256":"addb1f0e04e13cc66556e0a229af172672d1a54079cda563d96fa061f30538c3","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1917d-cdde-777b-9e4c-6caa1495d8d8"},"Hashes":{"key_sha256":"669eaca2922a922794cc66f21498dbb09bb98c2c6d5b42ec7e983818df0e74ed","cert_sha256":"70bd5bfc06daf6b2bee486aab5ae422fef4ae68420e45105accb279d453986d4","node_json_sha256":"c0843540cd6c7cf3e79175baadbf83d724e32863cbd528581670f52d3a79f788","usage_db_sha256":"addb1f0e04e13cc66556e0a229af172672d1a54079cda563d96fa061f30538c3","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a1917d-cdde-777b-9e4c-6caa1495d8d8"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- rerun_hashes_stable: true
- netbird_owned_socket: true
- shared_runtime_survives_consumer_stop: true
- restored_services: {"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}
