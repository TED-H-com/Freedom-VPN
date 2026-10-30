# lease-expiry

Control stop leaves forwarding off

Status: blocked

Run: 5de03570-0436-477d-9847-6f6984432bd8

Source: a7a712aab8a67fcc33514a9ac76cb877a90c8143

Spec: §5, §19 AT29

- table_before: true
- policy_before: drop
- member_accept_before: 0
- table_after: true
- policy_after: drop
- member_accept_after: 0
- note: the forward chain had no member accept rule before the stop, so AT29 is not claimed
- control_restarted: true
- owner: w2-agent-core: AT29 needs active forwarding before control loss
