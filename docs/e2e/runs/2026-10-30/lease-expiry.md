# lease-expiry

Control stop leaves forwarding off

Status: blocked

Run: 6bf270e4-abb1-43ff-93eb-18d282fc7c2a

Source: 5477de091f449047193fdc9114cfca4ced5b36e6

Spec: §5, §19 AT29

- table_before: true
- policy_before: drop
- member_accept_before: 0
- table_after: true
- policy_after: drop
- member_accept_after: 0
- note: the forward chain had no member accept rule before the stop, so AT29 is not claimed
- control_restarted: true

