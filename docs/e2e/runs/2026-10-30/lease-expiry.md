# lease-expiry

Control stop leaves forwarding off

Status: blocked

Run: 5852f09a-30e8-45bc-8983-b5f212e33640

Source: 5eb716e1a36ced17b8013646c9282c92746c81da

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

