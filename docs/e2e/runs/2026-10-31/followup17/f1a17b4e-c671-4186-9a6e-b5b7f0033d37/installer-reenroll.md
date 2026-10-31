# installer-reenroll

Controlled reinstall, retained identity and retired certificate refusal

Status: pass

Run: f1a17b4e-c671-4186-9a6e-b5b7f0033d37

Source: 6e3667f57a55c2757644b2840d4a6763435f5009

Spec: ENR-06, AT07, §6.3

- revoke: 202
- reenroll: {"Exit":0,"Status":"reenrolled","Code":"baseline","Incarnation":"01a190fd-93f0-7b36-810a-ac5e778c0353","NodeID":"01a190fa-44c8-7854-834a-485f52b7c598","RetiredMode":"700","RetiredHashes":{"key_sha256":"62982d4987588c8c0492c4858f18303f78bd1baae3118ab1803416fbcb0df736","cert_sha256":"d423a8be66a7176774fc7a782669db371f305271b41a876b852f29cb9093548f","node_json_sha256":"b831e309728761f3b868cef5d8eb1568e8912e4bc4c5f00d263f818a731548cf"},"BoundStateBefore":{"agent_db_sha256":"54248846259c5445aa64d69f42c59bdaf9902715cfae2e5bd1518ac23af23e34","agent_db_shm_sha256":"absent","agent_db_wal_sha256":"absent","desired_state_json_sha256":"3a4ea4fd9881685dd9e3a565087e8ceeb846db960bd59bff17fc31447e4e8d83"},"RetiredBoundState":{"agent_db_sha256":"54248846259c5445aa64d69f42c59bdaf9902715cfae2e5bd1518ac23af23e34","agent_db_shm_sha256":"absent","agent_db_wal_sha256":"absent","desired_state_json_sha256":"3a4ea4fd9881685dd9e3a565087e8ceeb846db960bd59bff17fc31447e4e8d83"},"Message":"identity retired under identity.retired; usage store kept","Before":{"key_sha256":"62982d4987588c8c0492c4858f18303f78bd1baae3118ab1803416fbcb0df736","cert_sha256":"d423a8be66a7176774fc7a782669db371f305271b41a876b852f29cb9093548f","node_json_sha256":"b831e309728761f3b868cef5d8eb1568e8912e4bc4c5f00d263f818a731548cf","usage_db_sha256":"809f44c0faabfc664f92952b6074092dc3f4254b52b42da9784ace73ca58d36f","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a190fa-2cfb-7d5d-be3c-e01dc9e92c4d"},"Hashes":{"key_sha256":"43f33850c8c5584fc2e2d05e7ecfe92edacaea4686e16102af3ecd54be7e9924","cert_sha256":"9843ec49af74a6a16c4299e0ea3e26a018e3507d5371034d0083c3912f04b0f6","node_json_sha256":"10bc669a16c1df0313ae77bffeb28da822f9f4218f10283f42c58eb7213bc7de","usage_db_sha256":"809f44c0faabfc664f92952b6074092dc3f4254b52b42da9784ace73ca58d36f","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a190fa-2cfb-7d5d-be3c-e01dc9e92c4d"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- retired_certificate_http: {"direct":"403","proxy":"403"}
- usage_retained: true
- bound_state_retired: true
- restoration: true
