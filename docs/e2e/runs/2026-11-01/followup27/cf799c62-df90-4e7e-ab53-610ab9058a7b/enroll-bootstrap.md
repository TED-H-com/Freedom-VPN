# enroll-bootstrap

Installer token, one bootstrap, NetBird join

Status: pass

Run: cf799c62-df90-4e7e-ab53-610ab9058a7b

Source: cc12bcd7f39314a37d0a9d58ae64d5c16f8640af

Spec: §6, §19 AT06

- token_length: 52
- secret_available: true
- node_id: 01a197a3-6e35-7ad4-b222-6a74240ce7b5
- setup_key_persisted: true
- bootstrap_again: BOOTSTRAP_CONSUMED
- signal_before: first run client: DeadlineExceeded dialing netbird.example.invalid with no port; failed to connect to the signalling server: dial context deadline exceeded
- signal_after: Connected
- peer_count: 1
- setup_key_count: 1
