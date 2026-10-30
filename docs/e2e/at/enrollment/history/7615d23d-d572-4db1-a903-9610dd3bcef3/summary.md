# Enrollment acceptance

Every required check must pass. Missing hooks keep the complete AT blocked.

| AT | Status | Decisive observation | Evidence |
|---|---|---|---|
| AT06 | blocked | Prior failed check requires explicit review | [AT06.json](AT06.json), [AT06.md](AT06.md) |
| AT07 | blocked | {'changed_on_main': 'devstack follow-up 12 PR #204 StartClonedIdentityFixture', 'clone': {'agent_spki_sha256': 'a5951566617a893589fa2c93276728c07367454fd3422cc1e48a91e50ff504ed', 'machine_id_sha256': '966f6cec585b885596dc34e40057e5e6cbbb41f4556656b4ca565eab84b41496', 'netbird_public_key_sha256': '1cd7c4d89c466d13387fa77f61efaf3679057bc08694cf692c372f63c848f75f'}, 'clone_stage': 'enrolled', 'conflicts': 0, 'endpoint_rows': 0, 'gap': 'StartClonedIdentityFixture.Observe read node_identity_conflicts and mapping_frozen; both stayed unset', 'mapping_frozen': False, 'owner': 'PKI/enrollment + node identity adapter', 'read_ok': True, 'request': 'ENR-08', 'same_identity': True, 'source': {'agent_spki_sha256': 'a5951566617a893589fa2c93276728c07367454fd3422cc1e48a91e50ff504ed', 'machine_id_sha256': '966f6cec585b885596dc34e40057e5e6cbbb41f4556656b4ca565eab84b41496', 'netbird_public_key_sha256': '1cd7c4d89c466d13387fa77f61efaf3679057bc08694cf692c372f63c848f75f'}, 'source_stage': 'enrolled', 'started': True} | [AT07.json](AT07.json), [AT07.md](AT07.md) |

See [RUN.md](RUN.md) and [requests.md](requests.md). Immutable attempts are in `history/`.
