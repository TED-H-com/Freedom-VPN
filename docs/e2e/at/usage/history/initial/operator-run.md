# Operator Runbook: Usage Metering & Fair-Use Acceptance (AT14–AT19)

## 1. Automated Verification in CI / Local Lab

1. Start the isolated `w4-usage` stack on port base `48000`:
   ```bash
   FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
   ```
2. Execute the acceptance suite (expected non-zero exit while product preconditions on `main` remain `blocked`, after writing `AT14.json`..`AT19.json` and `AT14.md`..`AT19.md`):
   ```bash
   make -f mk/at-usage.mk at-usage
   ```
3. Tear down the stack:
   ```bash
   FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000 make -f mk/e2e.mk e2e-down
   ```

## 2. Real Multi-Node & Real-Device Measurement Checklist (§19.5)

Once the upstream blockers documented in [requests.md](requests.md) are resolved on `main`:

- **AT14 (`forward_admission_v1`)**:
  - Enroll a published exit node and authorize a member device via NetBird.
  - Send measured upload/download payloads across `fv0` -> `eth0` and verify `nft -j list table inet fvpn` counters `u_<device_hex>` and `d_<device_hex>` match within ±1%.
  - Confirm WireGuard handshakes, blocked RFC1918/ULA packets, and DNS queries to `100.64.0.1:53` do not increment `u_*` / `d_*`.
- **AT18 (Multi-Tier Shaping & Host Reserve)**:
  - Run concurrent `iperf3` streams for a Tier-2 shaped member, a Tier-0 unshaped member, and host SSH/control traffic on `eth0`.
  - Verify `tc -s class show dev fv0` and `tc -s class show dev eth0` enforce `effective_rate = min(plan_rate_limit, fup_tier_rate, exit_per_member_limit)` while preserving `1:2` host reserve bandwidth.
