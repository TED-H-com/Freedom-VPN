# installer-wrappers

ENR-06 fixed runtime layout, sampled hashes and restoration

Status: pass

Run: abd7ce3a-7b3f-4fa3-8120-c11635971e2c

Source: 0b9fb5ebbff0d1618679b8eed6925b7aa9cc7daf

Spec: §6.3, ENR-06, AT07

- registered_release_0: {"architectures":["amd64","arm64"],"manifest_sha256":"16ed81b1b7ec04c6895c84e1e89a92cabfbec4cb11a096584115b1e0a9ba5b96","release_id":"01a19175-e341-792a-b34b-af9c102815b4"}
- registered_release_1: {"architectures":["amd64","arm64"],"manifest_sha256":"d5345215f2a292da4b7d8d3c5ce0a8a60fd8ee6530a9b1203dbfeb824c2db9a3","release_id":"01a19175-ee15-7a11-8e75-01a53cdd0a93"}
- initial: {"Exit":0,"Status":"installed","Code":"baseline","Incarnation":"01a19176-03c8-77ec-ab1e-8cc16897277a","NodeID":"01a19175-fd93-7037-86a1-b2ff221c1d75","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"","Before":{"key_sha256":"absent","cert_sha256":"absent","node_json_sha256":"absent","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"absent"},"Hashes":{"key_sha256":"15263b9018b7ff0fd83175a373be85ccb8e33273e3d1b76c5665541d3feca4be","cert_sha256":"587e9bd18376955cc246e6261f49cdd7abbae434a32f3cf36420a1eac155adc0","node_json_sha256":"a02615689b32fcd59fb7279a008e8a39c420fd3166e17071217dd7bf5dbd35d9","usage_db_sha256":"3e690e04344e7578585cd710e2866e6093a4b3367552c52a82eaa09bbf4349f3","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19175-e341-792a-b34b-af9c102815b4"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- first_poll_barrier_no_desired: true
- fresh_updater_no_work: true
- rerun: {"Exit":0,"Status":"already_current","Code":"baseline","Incarnation":"01a19176-03c8-77ec-ab1e-8cc16897277a","NodeID":"01a19175-fd93-7037-86a1-b2ff221c1d75","RetiredMode":"","RetiredHashes":{},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"already_current","Before":{"key_sha256":"15263b9018b7ff0fd83175a373be85ccb8e33273e3d1b76c5665541d3feca4be","cert_sha256":"587e9bd18376955cc246e6261f49cdd7abbae434a32f3cf36420a1eac155adc0","node_json_sha256":"a02615689b32fcd59fb7279a008e8a39c420fd3166e17071217dd7bf5dbd35d9","usage_db_sha256":"4c6c857676681d6619f202947c02a17eb0cc79186e6c0f94ece1fb6c9954e618","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19175-e341-792a-b34b-af9c102815b4"},"Hashes":{"key_sha256":"15263b9018b7ff0fd83175a373be85ccb8e33273e3d1b76c5665541d3feca4be","cert_sha256":"587e9bd18376955cc246e6261f49cdd7abbae434a32f3cf36420a1eac155adc0","node_json_sha256":"a02615689b32fcd59fb7279a008e8a39c420fd3166e17071217dd7bf5dbd35d9","usage_db_sha256":"4c6c857676681d6619f202947c02a17eb0cc79186e6c0f94ece1fb6c9954e618","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a19175-e341-792a-b34b-af9c102815b4"},"Services":{"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}}
- rerun_hashes_stable: true
- netbird_owned_socket: true
- shared_runtime_survives_consumer_stop: true
- restored_services: {"fvpn-agent.service":"active","fvpn-updater.service":"inactive","fvpn-updater.timer":"active","fvpn-watchdog.service":"active"}
