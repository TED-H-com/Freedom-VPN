# Review r8-d057: exit groups, resolver-document delivery, resolver admin access

Reviewed D-056, D-057, and D-061 plus §7, §12, and §13 on current `main`. Proofs are new `*_test.go` files. Failing proofs expect the secure behavior and stay red. No product code was changed.

`go vet -tags integration` is clean for `internal/nodes/httpwire`, `node/agent`, and `node/dns`.

## High

### Published resolver documents are never placed in desired state

Where: `internal/desiredstate/compiler.go:463-466` builds `Dns` with resolver addresses only. `internal/nodes/httpwire/document_inputs.go:18` (`ResolveDocumentInputs`) and `internal/nodes/resolver_documents.go:50` (`RecordIssuedResolverDocumentTx`) have no production caller. `docs/nodes/WIRING.md:430-445` says the compiler has no before-signing or after-persistence hook and that newly compiled states have no resolver-document reference.

Scenario: A DNS profile is published through the real publisher (`publishResolverProfile`). The owning node's signed desired state has no `resolver_document` (`desiredHasRef=false`, `Get()` returns "required JSON member is unspecified"). `GET /agent/resolver-documents/{published id}` is 404 `NOT_FOUND` "Resolver document not found." for the owning node and for a second enrolled node, with the same message. `docs/nodes/WIRING.md` says that 404 is the intended result until the hook is wired. The decision text describes delivery as live, and this gap is not one of the two follow-ups excluded from this review.

Impact: `node/agent/decide.go:285-287` passes `dns_policy` whenever the in-force desired state names no document. A published blocking profile therefore never reaches `node/dns.Apply`, and a health answer can still say the policy is in force. Authorization in `ResolverDocumentForNode` (`internal/nodes/resolver_documents.go:83-96`) cannot see a reference that was never written into an envelope or an issuance audit row.

Proof: `TestReviewD056_OwningNodeReceivesPublishedDocument`

```
review_d056_test.go:38: TODO(review): published resolver document was not delivered to the owning node: node=01a18af7-7c3e-7be0-9cd1-889b85aeb704 otherNode=01a18af7-7c7e-7905-bbd8-6743dd9987e2 refErr=required JSON member is unspecified ref={ContentSha256: DocumentId:00000000-0000-0000-0000-000000000000 Generation:0 ProfileRevisionId:00000000-0000-0000-0000-000000000000 SizeBytes:0} want={ContentSha256:5f868b7ea527bd548f76cd3150d389ac6c9f399705069fbc877b10903b44a3ae DocumentId:01a18af7-7c70-7c00-abcd-d748640e578c Generation:1 ProfileRevisionId:01a18af7-7c69-7463-87d6-db96d2f1a57b SizeBytes:348} ownerStatus=404 ownerCache="no-store" ownerType="application/json; charset=utf-8" ownerBody={"error":{"code":"NOT_FOUND","message":"Resolver document not found.","request_id":"01a18af7-7c7c-7e03-a8b9-92853650ad99","retryable":false,"details":{}}} otherStatus=404 otherBody={"error":{"code":"NOT_FOUND","message":"Resolver document not found.","request_id":"01a18af7-7c85-76f4-a861-9bcf531a4ebe","retryable":false,"details":{}}} desiredHasRef=false
```

Fix: In `CompileTx`, call `ResolveDocumentInputs` before `signDesiredState` (`compiler.go:474`) and `RecordIssuedResolverDocumentTx` after the envelope row is persisted, in the same transaction. Until that lands, do not treat a passed `dns_policy` check as evidence that a published profile is in force.

Retention, which this test cannot reach because nothing is issued: `resolver_documents.go:95` allows the current incarnation at `desired_config_generation` or that generation minus one, lifecycle `registered`, incarnation not revoked. An N-2 document and another node's document fail the same `NOT_FOUND` sentinel as a missing id (`resolver_documents.go:97-98`).

## Medium

### Resolver admin HTTP client follows redirects

Where: `node/dns/client.go:243-245`. `newHTTPClient` sets a timeout and leaves `CheckRedirect` nil, so `net/http` follows up to ten redirects. `Applier.call` (`client.go:219-222`) sets Basic auth and then calls `Do`. Load-time `parseAdminOrigin` (`node/agent/dnsconfig.go:106-128`) only sees the configured URL. SIGHUP calls `reloadResolvers` (`node/agent/engine.go:357-377`) and rebuilds this same client. The control client refuses redirects (`node/internal/controlclient/client.go:139` and `:154-156`).

Scenario: The configured admin URL is `http://127.0.0.1:<port>/` and answers 307. A second listener on loopback stands in for another origin so the proof sends no packets off-box. `POST /control/filtering/set_rules` with body `{"rules":["review-d061-rule"]}` is delivered to that listener, and `call` returns nil, so the apply is reported successful even though the configured admin did not accept the rules.

On a different host (`127.0.0.2`) `net/http` strips `Authorization`. On another port of `127.0.0.1` it forwards the Basic credential, because its redirect policy compares hostnames and ignores the port. Both are different origins.

Proof: `TestReviewD061_AdminClientRefusesOffOriginRedirect`

```
review_d061_test.go:21: TODO(review): resolver admin client followed a 307 off the configured origin: hits=1 err=<nil> authorization="" credential_forwarded=false body="{\"rules\":[\"review-d061-rule\"]}"
review_d061_test.go:24: TODO(review): resolver admin client followed a 307 off the configured origin: hits=1 err=<nil> authorization="Basic ZTJlOnB3LWQwNjEtcmVkaXJlY3Q=" credential_forwarded=true body="{\"rules\":[\"review-d061-rule\"]}"
```

`authorization` on the second line is Basic auth for `e2e:pw-d061-redirect`.

Fix: Set `CheckRedirect` to refuse every redirect, the same way `controlclient.refuseRedirect` does. Do that inside `newHTTPClient` so load and SIGHUP both get it.

### A health answer can report dns_policy passed before the named document is applied

Where: `node/agent/resolver.go:57-64` clears `dnsDocument` and returns when `e.revoked` is set, before fetch or `Apply`. `node/agent/decide.go:285-287` then passes `dns_policy` because no document flag is set and desired state is in force. `node/agent/attest.go:214` maps that pass to reason "policy in force". `node/agent/engine.go:418-429` calls `answerHealth` on the lease-revoked branches without `evaluate`. `answerHealth` (`attest.go:127-132`) re-runs `Decide` from the current flags when the probe is refreshed, and does not call `applyResolver`.

Scenario: A desired state names a resolver document.

- Revoked step: the document was applied once (`calls=1`). The next step has `e.revoked` set. `Apply` is not called again, `dnsDocument` is false, and the attestation is `passed` / "policy in force".
- Lease revoke before evaluate: `syncDesired` loads a new generation that names the document, then `markRevoked` and `answerHealth` run. `Apply` was never called (`calls=0`) and the attestation is still `passed` / "policy in force".

Proof: `TestReviewD056_HealthAnswerBeforeDNSApply`

```
review_d056_test.go:146: TODO(review): revoked step attested dns_policy passed ("policy in force") for document 01900000-0000-7000-8000-000000000056 with no further Apply (calls=1 dnsDocument=false)
review_d056_test.go:173: TODO(review): lease-revoked branch attested dns_policy passed ("policy in force") for document 01900000-0000-7000-8000-000000000056 before Apply (calls=0)
```

Fix: If the in-force desired state names a resolver document and `applyResolver` did not record that document, `dns_policy` must be `failed` or `not_run`. Call `applyResolver` before every `answerHealth`, including the revoked branches in `engine.go:399-429`, or refuse to attest `passed` while `dnsDocument` is false and the desired state names a document.

A forged `If-None-Match` value does not take this path. See the held section.

## Low

### A group that disappears during publication replace is a 500

Where: `internal/exits/groups.go:21-30`. The count of `member_groups` is not `FOR UPDATE`. The following `DELETE`/`INSERT` can hit the `exit_groups.group_id` foreign key. That error is not `*apierr.Error`, so `pkg/apierr/apierr.go:210-220` renders `INTERNAL_ERROR` and logs the raw error.

Scenario: Membership is one group. The publication transaction deletes `exit_groups` and inserts the same group. An `AFTER DELETE` trigger on `exit_groups` deletes that `member_groups` row in the same transaction, which is the foreign-key failure a committed `DELETE` of the group produces if it lands after the unlocked count and before the insert. The client receives 500. The transaction rolls back, so the previous membership row is still there (`membership` length 1). An unknown id without that race is 422 `VALIDATION_FAILED` and writes nothing; that assertion is in the same test and passed before the trigger.

Proof: `TestReviewD057_UnknownGroupAndDeleteRace`

```
review_d057_test.go:157: TODO(review): group removed between count and insert: status=500 body={"error":{"code":"INTERNAL_ERROR","message":"Internal error. Quote the request ID when reporting this problem.","request_id":"01a18af7-7fbe-7873-bb74-2b1942dc6152","retryable":false,"details":{}}} membership=[01a18af7-7fb0-7349-8805-d30cbeebb48d]
```

Fix: `SELECT ... FROM member_groups WHERE id = ANY($1) FOR UPDATE`, or map SQLSTATE `23503` on the insert to 422 `VALIDATION_FAILED` / "unknown group". Keep the rollback so a lost group cannot leave a partial set.

## What I tried that held

### D-057 membership and group creation

`TestReviewD057_ConcurrentPublicationFencing` passed. Two `PUT /admin/nodes/{id}/publication` calls with different `group_ids` and the same `If-Match` produced one 202. The other was 412 or 409. `exit_groups` had exactly one of the two ids.

`TestReviewD057_MembershipSurvivesUnpublishRevokeAndRepublish` passed. Omitting `group_ids` left the row in place. Revoke left it in place and `exit_count` stayed 1. `exit_count` counts attached exits, including unavailable ones (`internal/nodes/groups.go:64`, `MemberGroup` comment in `pkg/api/api.gen.go:3374`). A later `PUT` with `publication_intent=published` on the revoked node was not 202 and did not change the row. An empty `group_ids` list cleared the row and `exit_count` became 0. Nodes have a `forbid_delete` trigger (`db/migrations/00004_guards.sql:10`); there is no hard delete that could resurrect membership. Republish that omits `group_ids` keeps the rows on purpose.

`TestReviewD057_UnknownGroupAndDeleteRace` passed its unknown-id and duplicate-id assertions: both 422, and `exit_groups` stayed empty. The legacy `PATCH` writer (`internal/nodes/nodes.go:846-866`) is a second membership writer in the same node transaction. An unknown id there returns validation that echoes the submitted UUID and rolls back. That path was inspected and not re-tested.

`TestReviewD057_GroupCreateRaceAndNormalization` passed. Eight concurrent creates of the same code left one row and one 201; the rest were 409. Eight concurrent creates with distinct codes and one shared name left one row. The 409 comes from `INSERT ... ON CONFLICT DO NOTHING` against `UNIQUE(code)` and `UNIQUE(name)` (`internal/nodes/groups.go:33-36`). Code `RaceCode` was 422 or 400 (pattern `^[a-z0-9][a-z0-9_-]{0,63}$`). Names `Case Fold`, `case fold`, and ` case fold` are three groups. `GET /admin/groups?code=casefold3` returns the raw padded name. Create, list, and the stored row agree. Plan references are `exit_group_ids` UUIDs, so those name variants do not alias a plan. `TrimSpace` is only the emptiness check (`groups.go:25`); the stored name is the raw string.

### Member-facing responses (§7)

`TestReviewD057_MemberResponsesDoNotCarryGroupIdentity` passed. Surfaces and result:

- `GET /api/v1/agent/desired-state` for the enrolled node was 200. The body did not contain the attached group's id, code, or name, nor those of an unattached group created in the same test. `DesiredState` (`pkg/api/api.gen.go:2459`) has no group field. `DeviceBinding` (`api.gen.go:2541`) carries device, member, peer, and addresses.
- `GET /api/v1/member/exits` as that member was 200 and contained the exit UUID. It did not contain either group's id, code, or name. `ListMemberExits` (`internal/nodes/nodes.go:635-685`) uses group ids only in the `WHERE` clause. `MemberExit` is exit id, display name, region, availability, and note.
- `GET /api/v1/member/me`, `/member/devices`, and `/member/usage` in this httpwire composition are the not-implemented operation (`internal/httpapi/composition.gen.go:1003`, middleware maps that to "Operation is not implemented."). Those bodies did not contain the group identities. The members adapter is a separate composition (D-039). Its `MemberSelf` has `limits.exit_count`, which `internal/members/httpadapter/reads.go:197` fills with a count of published registered exits. `MemberDevice` has no group field. `MemberDetail.member_group_ids` is the admin member read (`reads.go:124`).
- As a member, `GET /admin/groups?code=` for a real code and for `nosuchgrp`, `GET /admin/nodes/{id}`, and `POST /admin/groups` were all 403 with the same error code and message for the known and the missing group. The role check is `internal/httpapi/middleware.go:300-308`, before the handler.

### Resolver document fetch (D-056)

`TestReviewD056_DocumentFetchAuthAndIdenticalNotFound` passed.

- An unknown document id and the published-but-unissued id are both 404 `NOT_FOUND` with the same message. The unknown-id response has `Cache-Control: no-store`. The handler sets `private, max-age=0` (`internal/nodes/httpwire/resolver_documents.go:19`). `Transport` then overwrites every `/api/v1` response with `no-store` (`internal/httpapi/middleware.go:144-146`). That is stricter than the contract. `WIRING.md` calls the overwrite a pending header seam.
- `DELETE FROM dns_resolver_documents` is rejected with `immutable record` (`db/migrations/00180_dns_publish.sql:17-19`). The row cannot disappear through SQL. If `DocumentByID` returns `pgx.ErrNoRows` after the issuance check, `resolver_documents.go:110-112` returns the same NotFound sentinel. That branch was not executed, because reaching it needs a source that hides a row the authorization query already accepted, and issuance itself is unwired.
- The visitor writes `doc.Canonical` with `buf.Write` and does not JSON-encode it again (`internal/httpapi/gen/server.gen.go:8408-8410`). No 200 body was observed, because the owning node is not issued the document. Size is bounded by `ValidateResolverDocument` (1 byte through `canonjson.MaxBytes`, 1 MiB on the column).
- After admin revoke, the same certificate gets 403 `IDENTITY_REVOKED` on the document and on desired state, with the same code and message (`internal/pki/service.go:54` when the fence denies the identity).
- A certificate issued by a different CA gets 401 `UNAUTHENTICATED` on both routes, with the same code and message. `node_certificates` rejects `DELETE` (`db/migrations/00110_pki_enrollment.sql:98`), so an enrolled certificate cannot be removed to simulate "signed here, not stored". The stored-miss path is `internal/pki/service.go:72-73` (`Certificate is not registered.`).

`TestReviewD056_EvidenceIsReportedWithoutIssuanceCheck` passed. Central accepted `outcome=applied` for a document id that was never stored, projected it on `NodeDetail.dns_policy_evidence`, and then replaced it with a later report whose generation was lower. `validateDNSPolicyEvidence` (`internal/nodes/dns_evidence.go:16-34`) checks unique feed codes and that a passed `dns_policy` check has `outcome=applied`. It does not check issuance or generation monotonicity. D-056 says central projects reported evidence and does not verify apply success, so this is held by that decision. A duplicate `feed_code` was 422. Thirty-three feeds were 422 (OpenAPI `maxItems: 32` on `DnsPolicyEvidence.feeds`). The service itself does not cap the feed count; the HTTP schema check does.

Agent verification that held:

- `TestReviewD056_TruncatedAndSizeClaimDoNotApply` passed. A body one byte short is `SIZE_MISMATCH`, detail "Resolver document size differs from the signed reference.", `Apply` calls stay 0, evidence outcome `failed`. A short body whose digest matches the bytes, with `SizeBytes` claiming the longer length, is also `SIZE_MISMATCH` and does not call `Apply` (`matchResolver` compares length before the digest, `node/agent/resolver.go:136-149`).
- Existing `TestResolver_DigestMismatch` covers a one-byte flip (`HASH_MISMATCH`, `Apply` not called). Existing `TestResolver_SizeOverflow` covers a longer body (`PAYLOAD_TOO_LARGE`).
- `TestReviewD061_MissingResolverStaysNotRun` passed. With a named document and a nil applier, the check is `failed` / `DEPENDENCY_UNAVAILABLE`, evidence outcome is `not_run`, and `serving` stays false (`resolver.go:88-90`).
- `TestReviewD056_ForgedETagDoesNotHideTheNewDocument` passed. `desiredETag` set to `"forged-etag-not-the-body"` did not 304. The next step replaced the ETag, loaded the new resolver reference, called `Apply` once, and only then attested `dns_policy` passed. A normal step applies DNS inside `evaluate` before `answerHealth` (`engine.go:442-449`).

### Resolver admin credentials (D-061)

`credentialMode` (`node/agent/dnsconfig.go:293-301`) requires a regular file, mode exactly `0600`, no setuid/setgid/sticky, and `syscall.Stat_t.Uid == 0`. Existing `TestResolverCredentialFileRefused` covers mode `0644` and a non-root `0600`, and checks the log does not contain the secret or the path. Existing `TestDNSConfigHostLocal` rejects a public IP, a DNS name, and an unassigned private address at load. Existing `TestResolverReloadClearsBadApplier` shows a bad file on reload nils `dnsApply` and sets the static detail. `reloadResolvers` (`dnsconfig.go:199-229`) does not keep the previous applier: failure calls `refuseResolvers`, which sets `dnsApply` to nil. SIGHUP does not re-read `agent.json`, so a public URL written to the config after start is not adopted.

`TestReviewD061_CredentialAbsentFromLogsObservationAndAttestation` passed. A `0644` credentials file containing `pw-d061-do-not-leak` made reload fail closed (evidence `failed`, attestation `dns_policy` not passed). The secret and the credentials path were absent from the structured log, the observation JSON, `health-attest.json`, and the check detail. The detail string is the static "Resolver credentials are not usable."

## Not reported again

Removing an exit group leaves bindings on nodes in the removed group, because batch recompile covers only the plan's remaining groups. `dns_policy_evidence` contract typing is assigned to contracts revision 6. The OpenAPI document already has a typed `DnsPolicyEvidence` on the observation report; this review did not re-open that.
