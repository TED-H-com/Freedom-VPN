## D-144 Nightly Stack Gate

- Verdict: **pass**
- Source commit: `9fbff94c84b0d29c2d78f420e76726f7c9a706ad`
- Run ID: `8a436a87-5efe-4c18-a7ec-72be170cb69b`
- Evaluated at: `2026-11-04T10:10:16Z`
- Counts: pass=32, expected_blocked=1, unblocked=0, unexpected_blocked=0, fail=0, error=0

### Expected-Blocked Rows (acceptance status remains `blocked`)

| Scenario | Acceptance Status | Owner | Reason | Detail |
|---|---|---|---|---|
| `member-ipv6` | **blocked** | w3-devstack-e2e / platform operators | Default native member container joins with --disable-ipv6 (DisableIPv6: true) and no IPv6 underlay or per-platform route/capture campaign is provisioned in the default devstack acceptance run (transport_scenario_test.go memberIPv6). |  |
