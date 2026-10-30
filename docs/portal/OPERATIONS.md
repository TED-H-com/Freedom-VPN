# Portal Live Stack Operations & D-050 Deviations (`OPERATIONS.md`)

`apps/portal/e2e/dev-stack.mjs` runs `fvpn-control` and `fvpn-worker` against an isolated PostgreSQL database (`$FVPN_TEST_DATABASE_URL`) and exercises real control-plane and agent APIs:

- **Member identity & device discovery (D-052):** Fake NetBird serves users and peers matching the dev OIDC subject (`is_service_user: false`, `is_blocked: false`); `fvpn-worker` runs `IdentityDiscovery` (`mapping_proof = shared_issuer_public_subject:<issuer>`) and projects `member_devices` and `device_authorization_observations`.
- **Node enrollment, desired state & update batch results (D-048, D-050):**
  1. Issues an enrollment token via `POST /api/v1/admin/nodes/enrollment-tokens` and enrolls `liveNode` via `POST /api/v1/agent/enroll`.
  2. Creates the fleet update batch via `POST /api/v1/admin/update-batches` (`scope: 'all_nodes'`), which sets `nodes.desired_release_id` on all fleet nodes.
  3. Pulls the signed desired-state envelope and update intent via mTLS `GET /api/v1/agent/desired-state` (which compiles `desired_states` and populates `policy_refs`).
  4. Delivers the signed-hash update result via mTLS `POST /api/v1/agent/update-results` (`state: 'succeeded'`, `policy_generation` matching the compiled desired-state generation) and confirms via `POST /api/v1/agent/observations`, then polls `GET /api/v1/admin/update-batches/{batch_id}`.

## Remaining D-050 Fixture Deviations

Until the dev stack has a second fake agent reporter, only two fixture seeds remain in `apps/portal/e2e/dev-stack.mjs`:

1. **`exits` publication state seed:** `UPDATE exits SET publication_intent = 'published', publication_observed = 'active'` on the enrolled node so member exit views and serving checks observe an active exit.
2. **`SEEDED_NODE_2_ID` fixture node:** Inserts a secondary offline node (`nodes`, `node_incarnations`, `exits` for `jp-tyo-01` / `arm64`) with no agent reporter so the real `POST /api/v1/admin/update-batches` call includes a second target in `waiting_online` (`unconfirmed: true`, `policy_generation: null`, `policy_refs: null`).
