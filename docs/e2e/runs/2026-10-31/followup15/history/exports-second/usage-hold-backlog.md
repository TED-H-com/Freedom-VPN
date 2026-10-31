# usage-hold-backlog

Usage production, stable restart snapshot and ACK release

Status: pass

Run: 630b9f07-62d2-4db1-9737-9060732db1f1

Source: 3c25ce9daa081874988c9b09bc8c26e66d3c1b46

Spec: §10.5, AT07, ENR-05

- produced_while_held: true
- acked_before: 9
- acked_held: 9
- unacked_held: 1
- restart_preserved_hashes_and_ack: true
- acked_after_release: 11
- released_caught_up: true
