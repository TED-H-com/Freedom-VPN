# Usage acceptance after serving

A member must first have an API-created plan, projected entitlement and
native device binding attached to a published exit. Use the same group ID in
the plan and publication request. Read the signed desired-state binding and
its interval before sending usage through the node mTLS endpoint. D-050
forbids hand-writing usage receipts, samples, device bindings or entitlements.

The existing usage replay check submits one bounded sample for a projected
binding, resends its exact bytes and verifies identical results plus the exact
member total. This validates acceptance/replay; it does not by itself prove
that a real flow was metered. The forwarding continuation must additionally
observe the agent's samples after lab-local member traffic.

See [usage evidence](../runs/2026-10-30/usage.md). When the D-057 group API is
unavailable, no serving binding is fabricated and no usage batch is sent.
Owner: `w3-wire-nodes` follow-up 11 for the serving prerequisite, then
`w3-wire-usage` for usage acceptance/projection. The record must distinguish
these prerequisites from an observed usage API failure.

## Follow-up 7

The runner now requires real sampled traffic: after a positive device admission
counter increment, it polls the member usage surface for the agent's uploaded
bytes. It no longer injects a 3000-byte fixture batch. Until the agent refreshes
its initial empty-member lease, there is no admitted flow and this scenario is
blocked by **w2-agent-core**, before w3-wire-usage sample acceptance. Synthetic
batch replay remains with the dedicated usage harness and is not substituted
for a native sampled flow here.
