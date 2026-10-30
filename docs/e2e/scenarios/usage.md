# Usage acceptance after serving

A member must first have an API-created plan, projected entitlement and native
device binding attached to a published exit. The plan and publication use the
same group ID. D-050 forbids hand-writing usage receipts, samples, bindings or
entitlements.

The runner requires a positive native admission counter increment, then polls
`GET /member/usage` for the running agent's sampled and uploaded bytes. It no
longer injects a 3000-byte fixture batch. This checks native sampling and arrival;
exact-byte reconciliation and replay remain separate usage-harness cases.

See [usage evidence](../runs/2026-10-30/usage.md). Follow-up 9 selects the
API-published NetBird exit explicitly, then proves a controlled-origin admission
counter increase before polling the usage API. The former empty-member lease
gate is fixed by agent follow-up 9 (#146); an unselected native exit was the
next lab-owned stop. Records identify sampler/API failures separately from a
missing counted-flow prerequisite.
