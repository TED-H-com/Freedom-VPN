# fixture-member-admission

Native identity, finite grant, selected exit and counted forwarding

Status: fail

Run: a7c41436-82b3-4665-b75c-e2887f2e1009

Source: d37720e9062b3ff4528ec0213d38c10d1b6309f3

Spec: §8, AT02, AT12

- grant: 202
- portal_issuer: https://netbird.example.invalid/oauth2
- portal_sub: [redacted]
- join: embedded IdP native user; exact peer user id compared with portal subject
- same_subject: true
- device_authorized: true
- member_netbird: netbird_up
- devices: 200 authorized 1 of 1
- device_authorized: true
- binding_revision: 1
- binding_valid_from: 2026-10-31T08:15:44Z
- binding_valid_until: 2026-10-31T09:15:29Z
- binding: projected
- counted_attempt: 1
- admission_counter: unavailable after 90 seconds
- heartbeat: v1
boot_id 97c018d7-a925-4723-9127-83c097de1c8a
mono_ns 18174592807588
wall_unix_ns 1793434637076436762
policy_expires_mono_ns 0
baseline_ok 1
serving 0
generation 1
reason safety_check_failed
