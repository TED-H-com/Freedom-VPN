# Follow-up 23 — 2026-11-01

- [Native router and pinned-client analysis](a26d9b0f-bd1b-48df-9546-4105f9b3f328/analysis.md): routed IPv6 and IPv4 both 64,064 bytes / 1 ms; one reactive IPv6 PTB and learned 1280 PMTU; IPv4 DF=0 transit fragmentation. Receive-only failures and F11-01 stay failed. Native campaign 14/0/3, exit 1.
- [DNS delivery and positive IDNA](9bb2098e-1c6f-48a8-9fbd-fd99bd9fc49e/analysis.md): RR UDP succeeds in 2 ms; IDNA UDP/TCP in 1 ms. Exact relay→node→member tunnel→local stub joins. Historical F12-04 SERVFAIL was not reproduced and remains unattributed. DNS campaign 11/0/1, exit 1, retaining the singular capture-statistics failure and separate same-capture readback.
- [Ordered gates and actual output tails](gates.md). All final offline gates pass, including whole-module pinned lint, evidence hygiene, handover and amd64/arm64 builds. Physical arm64 is not run.

Before the native run, all 155 follow-up 22 pins verified. They remain unchanged.
All 43 new private pcap receipts match exact byte count/SHA/mode 0600. All owned
containers, networks, volumes and images are removed; shared Postgres untouched.
No product file, member acceptance file, upstream pin or global MTU/MSS changed.
New immutable files are pinned in [the manifest](../../../followup23-captures.sha256).

Reproduce with a fresh `make -f mk/e2e.mk e2e-up E2E_RESET=1`, then
`FVPN_E2E_FOLLOWUP23=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup23Native$'`, then e2e-down.
Use a separate reset stack with `FVPN_E2E_TWO_RESOLVERS=true` for
`TestE2E_Followup23DNS`; always tear down. Respect the two-suite capacity rule.
Postprocess only new unpinned archives; never regenerate these files in place.
