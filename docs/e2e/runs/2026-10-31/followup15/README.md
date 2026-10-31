# Follow-up 15 evidence — 2026-10-31 UTC

The final implementation is `4fb5fa88ac400605ac08ee42c8eac99e936ec2a3`.
All runs use the assigned `e2e` prefix and port base `18000`, on disposable
lab networks. Addresses, accounts, keys and policy inputs are TEST ONLY.
The final evidence commit adds documentation and records only.

| Run | Source | Pass / blocked / fail | Scope |
|---|---|---|---|
| `2347f2a4-02aa-4f56-94f2-c3bde88b5a6d` | `4fb5fa8` | **32 / 1 / 0** | Final full acceptance, 863.78 s; records in this directory. Make exits 2 because IPv6 is blocked. |
| `12d10041-9d33-4fbb-a4b0-9d2a21bce785` | `4fb5fa8` | **21 / 0 / 1** | [Final exports](exports/), independent fresh stack, 387.05 s; Go exits 1 for the installer product failure. |
| `630b9f07-62d2-4db1-9737-9060732db1f1` | `3c25ce9` | 21 / 0 / 1 | [Second exports](history/exports-second/); 367.10 s. Installer product failure. |
| `be58af04-a09f-41c5-b369-05be49de8c0d` | `03839e5` | 22 / 1 / 1 recorded | [Interrupted preflight](history/acceptance-preflight/), not a completed full run. Stopped after the DNS fixture failure; later corrected through the loopback resolver client. |
| `299afccc-d98b-4bde-821e-6930bdbbfa69` | `9e493e8` plus uncommitted adapter | 1 / 0 / 1 | [Focused installer](installer-focus/); 27.36 s. Real enrollment succeeded, then service startup failed. |
| `926791b9-36ef-4051-9d44-cd7eafdc8792` | `9e493e8` plus uncommitted implementation | 18 / 0 / 3 | [First exports](history/exports-first/); 382.13 s. Relay proxy path, DNS client access and split enrollment listener issues were subsequently corrected. |

## Deciding observations

| Item | Evidence and limit |
|---|---|
| AT29 established transports | [Start/move/stop](established-transports.md): TCP and authenticated UDP replied; device bytes increased and then held at 3577 after cleanup. [Control loss](lease-expiry.md): member accept rules changed from 372 to 0; both senders remained active with rising attempts while replies and retained counters held. |
| AT31 forced relay | [Forced relay](forced-relay.md): connected path changed from 1 direct / 0 relayed to 0 direct / 1 relayed, then back. TCP and authenticated UDP worked on relay; the same sockets continued on direct. This is not the full relay capacity or physical platform matrix. |
| Exit-traversing DNS | [Exit DNS](exit-dns.md): UDP DNS upload/download increased 65/98 bytes and TCP 307/340 bytes, with zero Internet-counter deltas. Queries reached the product-managed filtering resolver through the selected exit. Native two-resolver fallback is still open. |
| IPv6 | [Member IPv6](member-ipv6.md) is blocked: native `DisableIPv6=true`, no tunnel IPv6 address, and no provisioned IPv6 underlay/platform capture matrix. No IPv6 no-leak claim. |
| ENR-05 usage hold | [Endpoint scope and snapshots](exports/usage-hold.md) and [real backlog](exports/usage-hold-backlog.md): ACK stayed at 9 while an unacknowledged event accumulated; paused snapshots were byte-identical, event hashes/ACK survived restart, and ACK advanced to 10 after release. This network-side POST 503 fault does not change the central ledger epoch. |
| ENR-06 installer | [Product finding](installer-finding.md): actual install exits 1 after real enrollment because the packaged NetBird unit cannot bind its default socket under its write restrictions. Successful rerun/newer-release/reenroll assertions remain unreached. No unit override or fake NetBird daemon substitutes for this failure. |

The historical scenario name `lease-expiry` does not establish which safety
deadline fired. Its rule cutoff was observed at `04:21:29Z`, before the signed
desired-state expiry of `04:30:50Z`; its last heartbeat still said `serving`.
This run proves control-loss fail-closed behavior on active sessions. It does
not isolate lease expiration or establish an exact cutoff SLA.

## Verification

On the final implementation, `make -f mk/e2e.mk e2e-lint`, `make go-lint`,
`go vet ./...`, `make go-build`, `go test ./tests/e2e/...`, and the tagged
ENR06/path/prefix checks all exited 0. Both linux/amd64 and linux/arm64 built.
The live run is amd64; physical arm64 execution remains operator work.

Final `e2e-down` exited 0. A separate ownership-filtered inventory found zero
remaining containers, networks, volumes or image tags for this prefix.
Formatting, whitespace, owned-path and prescribed public-hygiene checks passed.

See [RUN.md](../../../RUN.md#follow-up-15-transport-usage-and-installer-hooks)
for the exported API, fault semantics and reproduction commands. Dated answers
under all six harness request files identify the remaining DNS, IPv6,
platform, retention and fleet-population work. The node entrypoint retains
Docker `ip nat` deletion; D-079 classifier proof remains netlab's scenario.
