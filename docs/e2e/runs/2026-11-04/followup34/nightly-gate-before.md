## D-144 Nightly Stack Gate

- Verdict: **fail**
- Source commit: `ab0899e6799a4d992c87215c5a6705e7e2fa506f`
- Run ID: `af4c130b-5029-401b-bd0f-e8afaad3d37f`
- Evaluated at: `2026-11-04T06:28:06Z`
- Counts: pass=31, expected_blocked=1, unblocked=1, unexpected_blocked=1, fail=0, error=0

### Expected-Blocked Rows (acceptance status remains `blocked`)

| Scenario | Acceptance Status | Owner | Reason | Detail |
|---|---|---|---|---|
| `member-ipv6` | **blocked** | w3-devstack-e2e / platform operators | Default native member container joins with --disable-ipv6 (DisableIPv6: true) and no IPv6 underlay or per-platform route/capture campaign is provisioned in the default devstack acceptance run (transport_scenario_test.go memberIPv6). |  |

### Unblocked Rows (passed; remove from expected-blocked list)

| Scenario | Acceptance Status | Previous Owner | Previous Reason | Detail |
|---|---|---|---|---|
| `lease-expiry` | **pass** | w2-agent-core | Candidate fvpn-agent started by fvpn-updater during update-good stepStart runs boot-time recoverOnce() while parent fvpn-updater still holds /var/lib/vpn-node/update.lock, hitting UPDATE_BUSY and latching recoverBlocked=true (reason recovery_required, serving 0) in memory until node-restart restarts the agent, so countedFlow finds no device admission counters during the 90s readiness poll. |  |

### Unexpected Blocked Rows (not on expected-blocked list; fails nightly gate)

| Scenario | Acceptance Status | Detail |
|---|---|---|
| `update-good` | **blocked** |  |
