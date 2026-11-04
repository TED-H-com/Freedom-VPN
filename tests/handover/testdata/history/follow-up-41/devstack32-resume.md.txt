# Follow-up 32 resumed proof — 2026-11-03

The orchestrator authorized a new forced invocation because the original
`e22872b7-1112-4082-8e2b-235275f4a659` failed in admission plumbing before
reaching enrollment or the publisher. That original run and ordinary
`c52b760c-c289-4220-a722-6c5a059819b9` remain byte-for-byte unchanged.

## Admission and exact body evidence

Before starting a stack, the [dry Docker command](wrapper-dry-resume.json)
ran through the corrected wrapper at 23:20:51.010299Z–23:20:51.098110Z.
`docker version --format '{{json .}}'` exited 0; its complete 1149-byte stdout
parsed as one JSON object with Client and Server, and stderr was empty. All
admission stdout/stderr goes to a separate private file, including denials;
neither Docker stream receives admission text. No reservation was taken.

Commit `419c065cbf93b83b0114f84071fee62981d90d99` adds `body_sha256` to every
node write receipt, computed from the exact serialized request bytes. This
allows direct same-body comparison without storing request secrets. The
ordinary HTTP call still uses the same serialization and request behavior.
Synthetic tests compare the digest to the bytes received by a stub server.

## First resumed forced invocation: retained blocked proof

Invocation `af726b13-9f9c-4675-b81f-edd78d3a5c24` ran at
23:24:17.453976Z–23:24:39.910663Z on source `419c065c`, with both
`FVPN_E2E_FOLLOWUP32=1` and `FVPN_E2E_PUBLISH_STALE_ETAG=1`, after a fresh reset.
Its [original log](af726b13-9f9c-4675-b81f-edd78d3a5c24/go-test.txt) exits 1:
five scenarios passed and `publisher-retry-proof` was blocked. The actual
[publisher](af726b13-9f9c-4675-b81f-edd78d3a5c24/publish.json) passed. Both
writes recovered from real 412s using the same key/body, and operation
`df7375b2-765c-4e9c-895d-6809c3caad59` reached `succeeded` with observed
publication `active`. Member admission and populated serving were not run
because the proof check stopped the runner. This run remains blocked.

The proof incorrectly required `current_resource_version` on every conflict.
A single [diagnostic request](conflict-response-diagnostic.json), retained
separately at 23:25:55.279449214Z–23:25:55.2830024Z, sent the same capacity
body with a new key and `If-Match="0"`. It returned 412 and a valid JSON error
with `details={}`. There was no parser loss. Read-only product inspection
explains this: `internal/nodes/nodes.go:86–98` rejects zero in `ParseIfMatch`
before looking up the resource version. The recorder correctly omitted an
absent detail. This diagnostic was one rejected write, with no retry, and was
not another publisher invocation.

Commit `59cae103757567bb32feb6db71cc8bb89dbb5d6c` fixes the owned proof helper
to honor the requested recording rule, **current version when present**. Its
stub regression covers both empty details and a supplied decimal version,
verifies PATCH→GET→PATCH and the fresh ETag, and rejects a stale retry ETag.
The forced proof now requires exactly one conflict and one retry per command;
key/body identity, accepted 202, durable operation rows and all downstream
predicates remain required. The publisher retry implementation did not change.
No product file or expected-blocked entry changed. The orchestrator's
shared-helper exception authorizes one further forced invocation and a fresh
ordinary invocation after this fix; the first result is preserved.

The forced zero ETag proves handling of the real API's exact status/code. It
does not establish that an agent observation caused the conflict, nor exercise
the product's nonzero-version lookup branch. The earlier description of zero
as a valid stale version needs this qualification: OpenAPI's header pattern
allows zero, but the product parser rejects it before lookup. Supplied current
versions remain covered by synthetic tests. The original isolation 19 race
explanation remains plausible and unproven.

## Reservation during second stack setup

The second `e2e-up E2E_RESET=1` began after admission at 23:29:54.963150Z.
A reservation appeared during setup. The next nested Docker command was held
at 23:30:25Z and 23:31:25Z; its admission messages went only to the private
admission file. The bounded foreground wrapper checked every 60 seconds and
started that command only when admission returned `free` at 23:32:25Z. Setup
completed at 23:32:26.988290Z. No forced test ran during the hold, and no setup
retry or reservation of this lane's own was needed.

### Attempt receipts: `af726b13-9f9c-4675-b81f-edd78d3a5c24`

`PATCH /api/v1/admin/nodes/01a1a3ba-5fa4-7e43-bf4d-4b4f0b993dc9`

- Idempotency-Key (all attempts): `9bc52246-5cca-4dbb-b776-76d515b7bf48`.
- Body SHA-256 (all attempts): `4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d`.
- Terminal outcome: `accepted`; command status `pass`.

| Request | Started UTC | Finished UTC | Sent If-Match / returned ETag | Status / code | Current version detail |
|---|---|---|---|---|---|
| PATCH 1 | 2026-11-03T23:24:33.694262942Z | 2026-11-03T23:24:33.697786879Z | `"0"` | 412 RESOURCE_VERSION_CONFLICT | not supplied |
| GET re-read | 2026-11-03T23:24:33.697806052Z | 2026-11-03T23:24:33.70121898Z | `"6"` | 200  | — |
| PATCH 2 | 2026-11-03T23:24:33.701258287Z | 2026-11-03T23:24:33.706586484Z | `"6"` | 202  | not supplied |

`PUT /api/v1/admin/nodes/01a1a3ba-5fa4-7e43-bf4d-4b4f0b993dc9/publication`

- Idempotency-Key (all attempts): `d4814143-839b-459d-8cf7-f41407269b70`.
- Body SHA-256 (all attempts): `960d69301a7fefdc7a1b62122a83becb4abbb837fb284fc26730fbc4d697c534`.
- Terminal outcome: `accepted`; command status `pass`.

| Request | Started UTC | Finished UTC | Sent If-Match / returned ETag | Status / code | Current version detail |
|---|---|---|---|---|---|
| PUT 1 | 2026-11-03T23:24:33.709328596Z | 2026-11-03T23:24:33.712413024Z | `"0"` | 412 RESOURCE_VERSION_CONFLICT | not supplied |
| GET re-read | 2026-11-03T23:24:33.71241594Z | 2026-11-03T23:24:33.715408949Z | `"7"` | 200  | — |
| PUT 2 | 2026-11-03T23:24:33.715448105Z | 2026-11-03T23:24:33.720380291Z | `"7"` | 202  | not supplied |


## Second resumed forced invocation: pass

Invocation `95fb7eab-6795-48df-b3f2-ed014710d292`, source
`59cae103757567bb32feb6db71cc8bb89dbb5d6c`, ran at
23:33:05.102975Z–23:34:11.474516Z and exited 0: **8 pass, 0 blocked, 0 fail**.
Both required environment switches were `1`. The
[publisher receipts](95fb7eab-6795-48df-b3f2-ed014710d292/publish.json) show
exactly one real 412, one GET and one 202 retry for each of PATCH and PUT.
The sent zero ETags, fresh versions, keys, body digests and UTC times appear
below. The [proof](95fb7eab-6795-48df-b3f2-ed014710d292/publisher-retry-proof.json)
read back one durable completed 202 operation row for each key, confirming
that the rejected command was not replay-cached and no duplicate command was
persisted.

All downstream predicates passed. Publication operation
`2e780f6d-bf98-4aab-a38c-1a74d631bbed` succeeded; its agent policy, data-path
probe and NetBird route read-back steps succeeded, and observed publication
became `active`. Native identity matched the portal subject, the device was
authorized and its binding projected, the member selected the exit, and a
controlled origin request returned HTTP 200 with counted forwarding.
Counters reached 1113 upload / 1518 download bytes. The
[serving gate](95fb7eab-6795-48df-b3f2-ed014710d292/publisher-serving.json)
joined generation `6`, one eligible member, accepted report
`01a1a3c3-5886-718b-9a3f-62c80197dd17`, and populated lease
`88befb48-6047-4d3d-ac31-67ec8e0ecf74`; desired/applied/local/kernel generations,
forwarding and the fresh serving heartbeat met the unchanged predicates.
This is fixture evidence, not a timing or platform acceptance claim.

The earlier **real-API validation not exercised** gap is resolved by forced
invocation `95fb7eab-6795-48df-b3f2-ed014710d292`. This does not change either
historical failed invocation or close isolation's independent acceptance run.

### Attempt receipts: `95fb7eab-6795-48df-b3f2-ed014710d292`

`PATCH /api/v1/admin/nodes/01a1a3c2-6bf3-7717-8949-92c910b9e648`

- Idempotency-Key (all attempts): `ffcf1ca2-e4c3-4a49-91f6-b204e0bfa651`.
- Body SHA-256 (all attempts): `4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d`.
- Terminal outcome: `accepted`; command status `pass`.

| Request | Started UTC | Finished UTC | Sent If-Match / returned ETag | Status / code | Current version detail |
|---|---|---|---|---|---|
| PATCH 1 | 2026-11-03T23:33:22.147207301Z | 2026-11-03T23:33:22.152607852Z | `"0"` | 412 RESOURCE_VERSION_CONFLICT | not supplied |
| GET re-read | 2026-11-03T23:33:22.152628584Z | 2026-11-03T23:33:22.156070614Z | `"7"` | 200  | — |
| PATCH 2 | 2026-11-03T23:33:22.156129663Z | 2026-11-03T23:33:22.163591889Z | `"7"` | 202  | not supplied |

`PUT /api/v1/admin/nodes/01a1a3c2-6bf3-7717-8949-92c910b9e648/publication`

- Idempotency-Key (all attempts): `adb7e726-009a-4446-87f6-b7ed11165d0f`.
- Body SHA-256 (all attempts): `c4d9d611f9b20457ff0d19484498e9fcba5523b131a7000523cdf7105d417f01`.
- Terminal outcome: `accepted`; command status `pass`.

| Request | Started UTC | Finished UTC | Sent If-Match / returned ETag | Status / code | Current version detail |
|---|---|---|---|---|---|
| PUT 1 | 2026-11-03T23:33:22.16637406Z | 2026-11-03T23:33:22.170268652Z | `"0"` | 412 RESOURCE_VERSION_CONFLICT | not supplied |
| GET re-read | 2026-11-03T23:33:22.170271301Z | 2026-11-03T23:33:22.174119545Z | `"8"` | 200  | — |
| PUT 2 | 2026-11-03T23:33:22.17416509Z | 2026-11-03T23:33:22.18028777Z | `"8"` | 202  | not supplied |


## Fresh ordinary invocation after the proof fix: pass

Invocation `3ad796d4-b669-42d2-aadf-46e60146822a`, source
`59cae103757567bb32feb6db71cc8bb89dbb5d6c`, ran at
23:35:13.618785Z–23:36:20.163561Z and exited 0: **8 pass, 0 blocked, 0 fail**.
`FVPN_E2E_FOLLOWUP32=1`; `FVPN_E2E_PUBLISH_STALE_ETAG` was unset. The previous
stack was removed and this one started with `E2E_RESET=1`. The
[publisher](3ad796d4-b669-42d2-aadf-46e60146822a/publish.json) used ETags
`"7"` for PATCH and `"8"` for PUT, each accepted once with 202 and no conflicts.
Each command had one durable accepted operation row. Operation
`eabf4f33-c2c6-42f9-9832-1b885f3e050c` succeeded and publication became active.

Member admission and controlled HTTP 200 forwarding passed, with 1113 upload /
1518 download bytes counted. The
[serving gate](3ad796d4-b669-42d2-aadf-46e60146822a/publisher-serving.json)
joined generation `6`, one eligible member, accepted report
`01a1a3c5-4f6e-70e2-b18c-3504355f23fd` and populated lease
`1afb26c2-4436-42c5-893a-960158bf5763`. All unchanged serving predicates passed.

### Requests to orchestrator

2026-11-03: the missing live-proof request is resolved. Route the existing
AT-owned helper audit to its owners, then let isolation perform its independent
post-merge run. This lane has not rerun or closed isolation's 12 blocked checks.

2026-11-03: route the zero-header contract detail to the API owner for review.
`contracts/openapi.yaml:2793–2798,2990–2991` allows `"0"` in the header pattern
and describes a current-version detail on 412; `internal/nodes/nodes.go:86–98`
rejects zero before lookup and returns empty details. The retained diagnostic
establishes that behavior. No product or contract change was made here; the
client records the detail when supplied and uses the re-read ETag as required.

### Attempt receipts: `3ad796d4-b669-42d2-aadf-46e60146822a`

`PATCH /api/v1/admin/nodes/01a1a3c4-627d-792d-9ed9-60ff88a28373`

- Idempotency-Key (all attempts): `0e3d8b25-8ac9-443e-9c43-bb4ecdbea20d`.
- Body SHA-256 (all attempts): `4939d40789ac64dedacaf0576b0f088ab48bf3b256d2263b70a3aa36ef39b67d`.
- Terminal outcome: `accepted`; command status `pass`.

| Request | Started UTC | Finished UTC | Sent If-Match / returned ETag | Status / code | Current version detail |
|---|---|---|---|---|---|
| PATCH 1 | 2026-11-03T23:35:30.880272031Z | 2026-11-03T23:35:30.977893599Z | `"7"` | 202  | not supplied |

`PUT /api/v1/admin/nodes/01a1a3c4-627d-792d-9ed9-60ff88a28373/publication`

- Idempotency-Key (all attempts): `86824d24-2f79-405b-a76a-792fc7d39234`.
- Body SHA-256 (all attempts): `9a7b8df32f6b3c6f90a4a41b5f1e8917e35d6d6607e3182f99fb402fb3e2d8b8`.
- Terminal outcome: `accepted`; command status `pass`.

| Request | Started UTC | Finished UTC | Sent If-Match / returned ETag | Status / code | Current version detail |
|---|---|---|---|---|---|
| PUT 1 | 2026-11-03T23:35:30.981296205Z | 2026-11-03T23:35:30.988468327Z | `"8"` | 202  | not supplied |
