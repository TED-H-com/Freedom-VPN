## D-144 Nightly Stack Gate

- Verdict: **pass**
- Source commit: `f6edbda9909268106f7a8e08c26aab8254ae1481`
- Run ID: `c17589fa-7e48-4dee-b574-7e29aca1effa`
- Evaluated at: `2026-11-04T15:10:53Z`
- Counts: pass=32, expected_blocked=1, unblocked=0, unexpected_blocked=0, fail=0, error=0

### Expected-Blocked Rows (acceptance status remains `blocked`)

| Scenario | Acceptance Status | Owner | Reason | Detail |
|---|---|---|---|---|
| `member-ipv6` | **blocked** | w3-devstack-e2e / platform operators | Default native member container joins with --disable-ipv6 (DisableIPv6: true) and no IPv6 underlay or per-platform route/capture campaign is provisioned in the default devstack acceptance run (transport_scenario_test.go memberIPv6). |  |
