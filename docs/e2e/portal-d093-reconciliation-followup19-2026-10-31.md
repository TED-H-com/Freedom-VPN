# Portal D-093 reconciliation — follow-up 19, 2026-10-31

Compared the fifteen **D093 IDs**, not the earlier numbered taxonomy in
RESPONSES.md, against portal follow-up 18 at product base `33034048` (including #323/#326).
Sources: [portal audit](../portal/DEV.md#d-093-harness-audit--2026-10-31-follow-up-18),
[simulator](../../apps/portal/e2e/dev-stack.mjs),
[simulated apply](../../apps/portal/e2e/applied-state.mjs), and the shared
[devstack harness](../../tests/e2e). Read-only source reconciliation; no portal
code or production verifier changes. TEST ONLY observations below are not
production deployment evidence.

| Row | Agreement / disagreement | Shared harness comparison and boundary |
|---|---|---|
| D093-01 | Agree: open | Portal combines issued lease eligibility with its own forwarding flag. Devstack reads the genuine heartbeat, accepted observation and counted member flow (`serving.go`, `forward_test.go`); it does not write forwarding success. Those separate runs do not close portal's simulator row. |
| D093-02 | Agree: open | Portal supplies successful internet/meter/tunnel checks. Native devstack uses the agent's verifier and kernel counters. The bounded probe origin authenticates the issued probe; a TCP/UDP helper alone does not establish every safety check. |
| D093-03 | Agree: open | Portal's passing QoS/source checks are synthetic; lease failure overrides use the issued lease. Devstack reads tc, nft and agent reports. Namespace-restart failures must remain failed even if central's older row said passed. |
| D093-04 | Agree: open | Portal's nonlease errors and updater results remain vectors. Devstack retains the historical sealed failure despite exit 0. Follow-up 19 separately checks initial apply exit against the new durable outcome and preserves failed attempts. No empty-error substitution is made. |
| D093-05 | Agree: open | `identity_frozen` is a central receipt annotation, not a harness-written observation field. Portal discards the accepted response. Shared clone/freeze fixtures use real agent handling and read back identity state; that does not repair the simulator's ignored receipt. |
| D093-06 | Agree: retired within simulator scope | `appliedState.apply` stages the desired snapshot and swaps after DNS validation; observations and renewals use that snapshot. No SQL generation fallback remains there. Devstack's first-offer export additionally compares exact response digest to durable SQLite acceptance and a matching applied receipt. Simulator retirement does not mean durable/kernel verification. |
| D093-07 | Agree: open | Seed/current/last-good IDs and versions are simulator outcomes. Devstack builds signed real releases for both architectures and reads updater journal/current symlink. The regular baseline uses TEST ONLY boot-layout configuration; the installer campaign uses packaged units and root-layout archives. |
| D093-08 | Agree: open | Portal supplies ready/healthy, peer/version/backend and random runtime identities. Devstack uses real native peers and kernel boot identity. Its synthetic identity-provider/payment inputs remain fixtures; restarting a container is not a new kernel boot. |
| D093-09 | Agree: open | Portal's 12/10/2/1840/4096 usage values are fixed report data. `UsageLedgerSnapshot` reads production CLI events and ACKs; `HoldUsageAcknowledgements` returns a transport 503 without changing ledger rows. D-118 adoption may change DB/WAL bytes; origin canonical payloads, sequence/hash and ACK are the preservation predicate. File equality alone cannot prove nonzero accounting or retired-stream delivery. |
| D093-10 | Agree: retired within simulator scope | Portal fetches exact resolver bytes through mTLS and validates length/hash/feed metadata before swap; absent references remain null. Devstack uses the genuine agent and AdGuard read-back. The portal has not acquired signature/key-rotation/time validation or DNS kernel enforcement by retiring this metadata fabrication. |
| D093-11 | Agree: open | Portal supplies state/stage/health/times/checkpoints. Devstack's installer apply uses the genuine checkpoint and journal. **Ordinary devstack update scenarios still have the explicitly TEST ONLY `e2e/meter.json` adapter** (`cmd/fvpn-e2e-nodeconfig/main.go`); do not cite those as the packaged final-sample proof. |
| D093-12 | Agree: open | Portal inserts release/artifact/verified rows, fleet support IDs/generation and initial policy activation. Shared `BuildRelease` → `RegisterRelease` waits for worker verification; `ActivatePolicy` uses the API. Its isolated Docker origin works under its own routing/TLS setup, which does not make a localhost Playwright origin acceptable to the production fetcher. The proposed compiled fixture fetcher is a separate, explicitly scoped test transport; see RESPONSES.md. |
| D093-13 | Agree: open | Portal inserts projector-owned rollups to show precision/gap/estimated cases. Shared counted-flow scenarios ingest genuine agent events and wait for projection. Keep the >2^53 UI fixture labeled; it is not measured traffic. |
| D093-14 | Agree: open | Portal writes restore fences, false authorization observations, a pending device and notes/resource versions. Shared normal member/revoke/regrant hooks drive APIs/native provider changes; destructive restore scenarios are separately identified faults. These test-state writes are central fixture exceptions, not real agent evidence. |
| D093-15 | Agree: open as documented setup | Fake provider/billing inputs and discovery outbox events drive real workers rather than claim their completion. Devstack also has TEST ONLY IdP/payment inputs, but native NetBird and worker read-back. Provider realism and input provenance remain explicit. |

Result: **15 agreements, 0 disagreements; 13 open / 2 retired in the portal's
stated simulator scope.** No portal correction request is raised by this
reconciliation. The earlier devstack response's fifteen numbered categories
were a replacement-path inventory, not a row-for-row D093 audit. This document
supersedes that mapping only; archived runtime outcomes remain unchanged.

The follow-up 19 run table is [here](runs/2026-10-31/followup19/README.md).
Live upgrades and signed checkpoints remain distinct from these simulator
classifications; successful live evidence does not retire a portal audit row.
