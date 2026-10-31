# installer-wrappers

ENR-06 fixed runtime layout, sampled hashes and restoration

Status: fail

Run: a3605492-0c78-4bf1-8219-4108269a0788

Source: 7af2b395f6c6521fb26b894342de21701810e1f7

Spec: §6.3, ENR-06, AT07

- registered_release_0: {"architectures":["amd64","arm64"],"manifest_sha256":"4587eed99d5a3d3675f6152136c47935d2a3aa2edf91c66cb25e8cbc7a8b84a6","release_id":"01a190e8-30f6-7098-a464-2f80e77dbc43"}
- registered_release_1: {"architectures":["amd64","arm64"],"manifest_sha256":"5efeb59bd7b2f41f059a251ecbbe259a1f4ca5875f34a5490fab394f8f1eb903","release_id":"01a190e8-3ba4-7361-94d5-a4d77dedfe32"}
- initial: {"Exit":1,"Status":"","Code":"failed","Incarnation":"01a190e8-5139-75fc-999e-54120f85dbbd","NodeID":"01a190e8-4afa-7a0f-846c-35a81c02a318","RetiredMode":"","RetiredHashes":{"key_sha256":"","cert_sha256":"","node_json_sha256":"","usage_db_sha256":"","usage_wal_sha256":"","machine_id_sha256":"","current_target":""},"BoundStateBefore":null,"RetiredBoundState":null,"Message":"failed: starting fvpn-node-netbird-up.service failed: ing (auto-restart) (Result: exit-code) since Sat 2026-10-31 07:42:46 UTC; 3ms ago Process: 1427 ExecStart=/opt/vpn-node/current/fvpn-node-install netbird-up (code=exited, status=1/FAILURE) Main PID: 1427 (code=exited, status=1/FAILURE) CPU: 41ms ","Before":{"key_sha256":"absent","cert_sha256":"absent","node_json_sha256":"absent","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"absent"},"Hashes":{"key_sha256":"2b786ce52fc92c5274571369a710a3e6bf14fb2d6bfd96d8fa07ac26ae38040a","cert_sha256":"a9a2ef19085e36eb34e9e772fa7a328fae0adf566445fa4ad521ad65679fff12","node_json_sha256":"fb7e53e6b48f5624909f73d846b1a2c9a6ad952093abe6f9fd1e8083f7870ff8","usage_db_sha256":"absent","usage_wal_sha256":"absent","machine_id_sha256":"186cc62b3ea133e7b6da9685430b24e54bb999ea093280f231a8b358af70ce92","current_target":"releases/01a190e8-30f6-7098-a464-2f80e77dbc43"},"Services":{"fvpn-agent.service":"inactive","fvpn-updater.service":"inactive","fvpn-updater.timer":"inactive","fvpn-watchdog.service":"inactive"}}
- error: <nil>
