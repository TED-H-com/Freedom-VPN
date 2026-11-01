# Follow-up 27 predeclared campaign

2026-11-01, TEST ONLY. Recorded before any new stack starts. Follow-up 26's
987 capture pins verified; its archive and manifest remain immutable.

- One fresh populated-hold fixture: at least two real granted members, retained
  populated generation g, real capacity PATCH, desired g+1, matching renewal
  responses withheld for 45 seconds. Observe alarm, one central escalation,
  then matching-lease recovery. Consumer acceptance matrices remain theirs.
- Eight fresh native transport stacks: IPv4 attempts 01–04 and IPv6 attempts
  01–04. Each retains one first-selection request after proven serving and one
  90-second UDP stream at 20 ms intervals on that serving node. Initial failure
  is retained; stream admission is checked independently. No replacement run
  silently substitutes for a failed prerequisite or request.
- Capture both ends and origin before first selection; retain passive capture,
  member-only tracing, endpoint/status/route samples, and per-send timestamps.
  No node tracing, transport repair, reconnect, NAT/kernel change or pin change.
- Fresh stacks run serially. Check other suite count before and after startup;
  wait while two other suites are active and record actual wait duration.
- Keep every attempt, including setup failures. Any additional diagnostic must
  be declared separately before it starts. Attribute only joined fresh evidence
  per underlay family. Historical F11-01, AT02 and F14-01 stay as recorded.

Lease-path source audit: `internal/nodes/httpwire/state.go:GetDesiredState`
returns only `DesiredStateEnvelope`; `RenewServingLease` returns
`ServingLeaseEnvelope`. `node/agent/sync.go` consumes them independently.
Therefore the transport seam targets only successful POST responses from
`/api/v1/agent/lease-renewals` whose signed generation matches the new desired
generation. Prompt 503 responses let Steps and real reports continue. Withheld
envelopes are discarded, never cached or replayed. Release forwards the next
genuine central-issued matching response byte-for-byte; joins distinguish all
withheld lease IDs from the delivered lease ID.
