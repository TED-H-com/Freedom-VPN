# Freedom VPN Portal (`apps/portal`)

`apps/portal` is the single-page web application (TypeScript strict, React 19, Vite 6) serving both the **Admin Console** (`/admin/*`, §18.1–§18.3) and the **Member Center** (`/member/*`, §18.4–§18.5).

Per `AGENTS.md` §2 and Project Spec §18.4, the Portal is an untrusted browser client:
- It never holds a NetBird management API token, release/policy/lease Ed25519 private signing key, or raw payment card data.
- Traditional Chinese (`zh-TW`) is the default UI language, with English (`en`) as the secondary language.

---

## Architecture & Contract Dependency

### 1. Single Source of Truth (`@contracts/schema`)
- `contracts/openapi.yaml` is the single source of truth for the 40 `/api/v1` endpoints, and `contracts/ts/schema.d.ts` is generated from it.
- `apps/portal/tsconfig.json` and `apps/portal/vite.config.ts` map `@contracts/*` to `../../contracts/ts/*`.
- `src/api/types.ts` imports `components`, `paths`, and `operations` from `@contracts/schema` and defines schema aliases (`NodeDetail`, `MemberSelf`, `OperationResource`, `ApiErrorCode`, etc.). **No hand-written `/api/v1` wire types exist in `src/api/types.ts`.**

### 2. Separation of Wire Types, Browser Auth Routes, and View Models
| Module | Responsibility |
|---|---|
| `src/api/types.ts` | Generated OpenAPI contract types and schema aliases from `@contracts/schema`. |
| `src/api/auth.ts` | Browser session routes (`/auth/session`, `/auth/login`, `/auth/logout`) that live outside `/api/v1` and are not part of the 40-endpoint OpenAPI schema, plus `__Host-fvpn-csrf` cookie reading. |
| `src/api/view-models.ts` | Portal-only UI view models and pure projection functions (`AdminOverviewViewModel`, `FupPolicyViewModel`, `DnsPolicyViewModel`, `deriveNodeSummaryPhrase`, etc.). Clearly separated from HTTP wire types. |
| `src/api/client.ts` | Operation-typed HTTP client (`FreedomVpnApiClient`). Every request is typed by path and HTTP method against `paths`/`operations` from `@contracts/schema`. |
| `src/api/errors.ts` | `ApiErrorCode` derived from `components['schemas']['ErrorCode']`, error envelope normalization, secret/SQL/stack-trace redaction, and `Retry-After` parsing. |
| `src/api/operations.ts` | Polling and step-status helpers for `202 Accepted` asynchronous `Operation` resources (§14.8). |

### 3. Operation-Typed HTTP Client (`src/api/client.ts`)
- **Compile-time safety:** `apiClient.callOperation({ method, path, pathParams, query, body })` extracts allowed paths, path parameters, query parameters, request body, and response body directly from `paths` and `operations` in `@contracts/schema`. Passing an unknown path or wrong request body is a TypeScript compile error.
- **Session cookies:** All requests send `credentials: 'include'`.
- **CSRF double submit:** Unsafe methods (`POST`, `PATCH`, `PUT`, `DELETE`) read `__Host-fvpn-csrf` from `document.cookie` at request time and attach `X-CSRF-Token`. Safe `GET` requests never attach `X-CSRF-Token`.
- **Idempotency (`§14.5`):** Mutating methods automatically attach a UUID `Idempotency-Key` scoped to the logical user action so retries reuse the same key.
- **Optimistic concurrency (`§14.5`):** `ETag` headers are captured on reads and sent as `If-Match` on `PATCH /admin/nodes/{node_id}`, `PUT /admin/nodes/{node_id}/publication`, `POST /admin/nodes/{node_id}/revoke`, `DELETE /admin/nodes/{node_id}`, `PATCH /admin/members/{member_id}`, and `POST /admin/policies/{definition_id}/activate` (`412 Precondition Failed` surfaces `ConflictDialog`; `428 Precondition Required` surfaces a reload prompt).
- **Large integers (`§10.4`, `§14.1`):** All byte counters, bps capacities, generations, revisions, and resource versions stay decimal strings on the wire and in React state, converting to `BigInt` via `src/lib/decimal.ts` only for arithmetic and comparison—never `Number` or `float64`.

### 4. Production Bundle Isolation vs. Contract-Backed MSW Mocks (`src/mocks/`)
- Production builds (`npm run build`, `import.meta.env.PROD === true`) tree-shake out MSW, mock handlers, and the top-bar persona switcher (`src/main.tsx`, `src/context/SessionContext.tsx`, `src/components/layout/AppLayout.tsx`).
- Mock mode is enabled **only** in non-production environments when `VITE_API_MOCK=1` (or `MODE === 'test'` in Vitest).
- `src/mocks/data.ts` imports canonical valid JSON fixtures directly from `contracts/examples/valid/*.json`.
- `src/mocks/handlers.ts` serves responses derived from those fixtures so mock responses cannot drift from `contracts/openapi.yaml`.
- `src/mocks/__tests__/contract_examples.test.ts` verifies every imported example's required keys, decimal-string fields (`^(0|[1-9][0-9]*)$`), UTC RFC 3339 timestamps ending in `Z`, and enum values against the generated union members.

### 5. Live Control Plane Dev Stack & Playwright E2E (`apps/portal/e2e/`)
- `apps/portal/e2e/dev-stack.mjs` (`make -f mk/portal.mk portal-dev-stack`) provisions an isolated disposable PostgreSQL database from `$FVPN_TEST_DATABASE_URL` (and drops it on exit), runs forward migrations (`fvpn-control -migrate-on-start`), starts real `fvpn-control` and `fvpn-worker` binaries with test OIDC, fake NetBird, and fake Stripe + HTTPS CONNECT proxy, enrolls a live mTLS exit node, and serves the production portal bundle on `http://127.0.0.1:4173`.
- `apps/portal/e2e/portal.spec.ts` (`make -f mk/portal.mk portal-e2e`) runs Playwright end-to-end tests against the live stack covering AT01–AT04, AT06–AT07, AT23–AT24, 412 `If-Match` conflict handling, policy activation, and audit event inspection.
- **412 Optimistic Concurrency Testing Rule:** A `412 Precondition Failed` test must always wait for the post-save (or post-reload) `GET` refetch to settle—and for the page to hold the settled `ETag`—before bumping `resource_version` out-of-band to make the `ETag` stale. On the page side (`AdminNodeDetailPage.tsx`), the visible "saved" banner (`saved-pending-apply-banner`) renders only after `queryClient.invalidateQueries` finishes refetching the resource (and immediately updates the cached `ETag` when the write response carries one), while the Playwright test registers `page.waitForResponse` for the post-save `GET` before clicking save and awaits both the response and the rendered `ETag` before invoking `/__e2e/bump-node-version` or a concurrent API write. This prevents an in-flight post-save `GET` from completing after the out-of-band bump and silently refreshing the form's `If-Match` header.

### 6. Operator Node Detail Notes: Provisional Lease Waiting Copy & Identity Conflict (D-067, D-071, D-072)
- **Provisional lease waiting state (`awaiting_member_grants` / `AWAITING_MEMBER_GRANTS`, D-067, D-071):**
  - During the AT09-style provisional enrollment/publish window—when a node holds a valid signed `ServingLease` with `members: []` while desired `device_bindings` is non-empty—`AdminNodeDetailPage.tsx` renders `forwarding_block_reasons` containing `awaiting_member_grants` and the `lease`, `qos`, and `source_binding` checks carrying `error_code: "AWAITING_MEMBER_GRANTS"` as a **waiting state** (`badge-warning` / `banner-info`), never as a failure (`badge-danger` / `failed`):
    - `zh-TW`: `等待會員授權；節點會自動續租（最長 60 秒）`
    - `en`: `Waiting for member grants; the node renews automatically (within 60 s)`
  - True lease expiration (`lease_expired` / `LEASE_EXPIRED`) retains its failure badge (`badge-danger`) and failure copy (`節點服務租約 (serving lease) 已過期，已停止接受會員流量。` / `Node serving lease has expired; member forwarding has stopped.`).
- **Node identity conflict projection (`NodeDetail.identity_conflict`, D-072):**
  - Absent (`undefined`) or `null` (for example on an offline node that has not yet submitted an mTLS observation): nothing is rendered (no placeholder text).
  - `{ mapping_frozen: false, conflict_count: 0 }`: renders a quiet line in the node identity header (`無身分映射衝突 (no identity conflict)` / `No identity conflict`).
  - `mapping_frozen: true`: renders a prominent `badge-danger` badge (`身分映射已凍結` / `Identity mapping frozen`), `conflict_count`, `last_conflict_at` formatted in the operator's locale with UTC, and the resolution link (`請撤銷此節點並重新註冊；系統不會自動解除凍結` / `Revoke this node and enrol it again; there is no automatic unfreeze`) pointing directly to the existing `#revoke-node-identity-btn` action. No unfreeze button exists, and endpoint hashes or addresses are never shown.

---

## Verification Commands

From `apps/portal`:
```bash
npm ci
npm run typecheck
npm run lint
npm test
npm run build
npm run format:check
npm run e2e
```

From the repository root:
```bash
make -f mk/portal.mk portal-verify
make -f mk/portal.mk portal-dev-stack
make -f mk/portal.mk portal-e2e
```
