# Follow-up 9: installer hand-off

Run `89fc6769-0a09-403d-ae88-84f736ee1dce` uses product main `5575470587a4e3014bd515e624ba2ad0aacbd140` and harness `5d98ea8f613fc415897b0a42940f0af6834ee2ae`. The installer extension is **blocked**. Its missing-desired-state, kernel boot identity, staging and tamper checks pass; healthy upgrade and central promotion do not. [Raw command evidence](installer-handoff.json), [requests](requests.md), [regular AT results](summary.md).

The installed sequence-10 release is `01a190a9-e3c2-7ca8-95d4-6ce5c777bfdf`; sequence 11 is `01a190aa-0968-78f3-ab78-d6ddeb6b8d2d`; sequence 9 is `01a190aa-1d38-7d6f-b62b-fb160834225b`. All are real signed amd64/arm64 bundles, registered and made eligible through the API.

| Command / stimulus | Status and exit | Journal / current / central deciding observation |
|---|---|---|
| `fvpn-node-install install --json …` through `RerunInstaller` | Exit 1 | Enrollment and installed layout retained; real NetBird unit cannot bind `/var/run/netbird.sock` on its read-only filesystem. Installer follow-up 3 remains a blocker. |
| Standalone `fvpn-updater seed --config … --manifest … --tree …` | Exit 1, `HASH_MISMATCH` | Live tree and verifier extraction differ under different umasks. No baseline fabricated. |
| Same seed with packaged `umask 077` | Exit 0 | `seeded_baseline` contains sequence 10, manifest/tree hashes, timestamp and kernel boot ID. `last_good_release_id` is empty and health null: seed is not a health pass. |
| Fresh `systemctl start fvpn-updater.service` with desired/pending absent | `Result=success`, `ExecMainStatus=0`, inactive afterward | No updater forwarding latch. This is D-080's no-work behavior on the tested main SHA. |
| Second fresh `fvpn-updater run --config …` | Exit 0 | Still no pending or desired state, no update job or new transaction. |
| `fvpn-node-install check --json` after no-work | Exit 2, drifted | No updater-service failure; stopped/failed member units and missing `/run/vpn-node` remain drift. Full healthy installation is not claimed. |
| `run` with a different legacy configured `boot_id` | Exit 0 | Config accepted; kernel ID still wins. Generated config omits the field. Seed, job claim and persisted lock-holder metadata carry the same kernel ID. |
| `upgrade --json` to sequence 11 with automatic apply held | `staged`, exit 0 | Pending record present, retained signed envelope path and fresh request UUID recorded; current remains sequence 10. |
| `fvpn-updater apply --pending …` | Exit 0, but local outcome **failed** | Job `9d496b3d-746b-5768-80a3-5397d1f9ec70`, source `installer`, requested sequence 11. Transaction seals `rolled_back`, `CHECKPOINT_FAILED` at `final_sample`. The meter barrier is [absent](installer-diagnostics.json). Current and unit start timestamps stay unchanged; pending is consumed. No switch, restart or health pass is claimed. |
| Subsequent `run`, then `flush` | Run exit 2; flush exit 0 with recorded rejection | Same terminal job remains; failed update keeps forwarding blocked. This differs from the fresh-node no-work case. |
| Product sender, then keyed exact submission/replay | HTTP 422 `VALIDATION_FAILED` | Both keyed requests report “Pre-health failure requires null health, an error and the unchanged attempt release”. Central current is null; updater current is its verified seeded baseline. No installer receipt or `source=installer` summary is accepted. |
| Stage again, alter one archive byte, `apply --pending` | Exit 1, `HASH_MISMATCH: staged artifact` | Rejected at verify; current unchanged, source `installer`, no switch. Same hash rejection class as regular AT25. |
| `upgrade --json` from sequence 10 to sequence 9 | **Unexpected** `staged`, exit 0 | Pending exists; expected `DOWNGRADE`/exit 1 does not occur. Harness preserves and withholds that pending record from the updater. |

The first agent poll is held by a runtime systemd condition for the fresh-node check. After releasing it, real serving evidence reports `member_forwarding_enabled=false`, reasons `not_published` and `safety_check_failed`, with no updater guard reason. A second runtime condition lets staging be inspected before apply; product `ExecStart`, stop/start commands, health checks and durable state remain the product's. The initial signed archive is retained for offline seed. The generated config's empty result endpoint is recorded before adding the owned lab endpoint and its CA/origin settings.

The installer node is retired through the API and read back as retired before the regular all-node snapshots. The regular node and later second node use a minimal kernel-boot adapter because devstack still generates a random agent boot UUID. The first diagnostic run that exposed the mismatch is [historical](history/follow-up-9-first-run/README.md). All current regular pass claims must come from the unified current run.
