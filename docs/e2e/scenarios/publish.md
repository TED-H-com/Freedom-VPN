# Exit publication — D-055/D-057

First call the [serving baseline helper](../RUN.md), then issue a fresh
installer enrollment. This exercises the one-off NetBird setup key and its
node correlation group; poll the product's incarnation peer projection.
No peer ID or readiness row may be supplied by the harness.

D-057 requires `POST /api/v1/admin/groups`, a plan using that group's ID in
`exit_group_ids`, and `PUT /admin/nodes/{id}/publication` with `group_ids` set
to that group and the current node ETag. Poll the returned operation and the
observed publication state. A 202, ready node or successful policy ACK alone
is not active publication.

The [publish record](../runs/2026-10-30/publish.md) contains exact HTTP status,
stored node/observation values, peer presence, exit-group count and operation
steps. The [catalogue record](../runs/2026-10-30/serving-catalogue.md) probes the
group API. `501 NOT_IMPLEMENTED` blocks Part B on `w3-wire-nodes` follow-up 11.
Do not work around it with an `exit_groups` or published-exit insert.
