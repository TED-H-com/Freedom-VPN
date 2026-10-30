# Native member forwarding — D-052/D-057

Use one public-subject issuer for Portal and native NetBird login. Create the
member's plan and exit group through the product APIs, grant or pay the member
through the product paths, then admit the real native device. Poll the
production identity mapping, device authorization and binding projections.
Match the plan's group to the published exit's `group_ids`.

Read the desired-state device binding before attempting a flow. Authorized
device projection alone does not establish forwarding; neither does an `fvnb0`
accept rule. The serving policy must contain `fvmb0` member accepts and a real
lab-local flow must traverse the exit. Lease expiry requires a successful
positive flow before control loss, then measured withdrawal after the lease
expires.

See [member traffic](../runs/2026-10-30/member-traffic.md),
[forwarding](../runs/2026-10-30/forwarding.md) and
[lease expiry](../runs/2026-10-30/lease-expiry.md). Until D-057's group/publication
API is available, these remain blocked on `w3-wire-nodes` follow-up 11; the lab
must not claim them from a synthetic published row. Resolver enforcement also
requires the D-056 central-to-agent delivery path; a central feed document is
not an on-node ACK.
