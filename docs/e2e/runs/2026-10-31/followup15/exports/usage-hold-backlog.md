# usage-hold-backlog

Usage production, stable restart snapshot and ACK release

Status: pass

Run: 12d10041-9d33-4fbb-a4b0-9d2a21bce785

Source: 4fb5fa88ac400605ac08ee42c8eac99e936ec2a3

Spec: §10.5, AT07, ENR-05

- produced_while_held: true
- acked_before: 9
- acked_held: 9
- unacked_held: 1
- restart_preserved_hashes_and_ack: true
- acked_after_release: 10
- released_caught_up: true
