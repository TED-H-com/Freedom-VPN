# Usage acceptance after serving

A member must first have an API-created plan, projected entitlement and native
device binding attached to a published exit. The plan and publication use the
same group ID. D-050 forbids hand-writing usage receipts, samples, bindings or
entitlements.

The runner requires a positive native admission counter increment, then polls
`GET /member/usage` for the running agent's sampled and uploaded bytes. It no
longer injects a 3000-byte fixture batch. This checks native sampling and arrival;
exact-byte reconciliation and replay remain separate usage-harness cases.

See [usage evidence](../runs/2026-10-30/usage.md). The current member has a real
desired-state binding, but the agent retains its initial empty-member lease.
There is no admitted flow. This scenario is blocked by **w2-agent-core lease
refresh**, before w3-wire-usage sample acceptance. The record distinguishes
that prerequisite from an observed usage API failure; the member usage API
itself answers 200.
