# Adversarial Review `r8-d057-review`: Exit Groups, Resolver-Document Delivery, and Resolver Admin Access (`D-056`, `D-057`, `D-061`)

- **Branch:** `r8-d057-review` (commit `c6ea8e7`)
- **Date:** 2026-10-30 (`2026-10-30 00:01:47 -0400`)
- **Reviewing model:** Not stated in `.agent/REVIEW.md` or commit `c6ea8e7` (commit had no `Agent:` trailer)
- **Decision / scope reviewed:** `D-056` resolver-document delivery, `D-057` exit groups and publication membership, and `D-061` resolver admin access (`§7`, `§12`, `§13`; `internal/desiredstate`, `internal/nodes`, `internal/exits`, `node/agent`, `node/dns`)
- **Adopted proof files on `main`:**
  - [`internal/nodes/httpwire/review_d056_test.go`](../../internal/nodes/httpwire/review_d056_test.go) (adopted in `#121`)
  - [`internal/nodes/httpwire/review_d057_test.go`](../../internal/nodes/httpwire/review_d057_test.go) (adopted in `#121`)
  - [`node/agent/review_d056_test.go`](../../node/agent/review_d056_test.go) (adopted in `#126`)
  - [`node/agent/review_d061_test.go`](../../node/agent/review_d061_test.go) (adopted in `#126`)
  - [`node/dns/review_d061_test.go`](../../node/dns/review_d061_test.go) (adopted in `#120`)

## Findings

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | Published resolver documents in `internal/desiredstate/compiler.go`, `internal/nodes/httpwire/document_inputs.go`, and `internal/nodes/resolver_documents.go` are never placed into signed desired state or recorded in `node_issued_resolver_documents`, causing `GET /api/v1/agent/resolver-documents/{document_id}` to return `404 NOT_FOUND` for the owning node. | closed | #109, #121, #127 |
| Medium | The resolver admin HTTP client in `node/dns/client.go` (`newHTTPClient`) leaves `http.Client.CheckRedirect` nil, following up to 10 redirects off the configured host-local origin and forwarding Basic auth credentials across same-host port redirects. | closed | #120 |
| Medium | Health answers in `node/agent/resolver.go`, `node/agent/decide.go`, and `node/agent/engine.go` can attest `dns_policy: passed` ("policy in force") on revoked or lease-revoked steps before `applyResolver` has applied the named `resolver_document`. | closed | #126 |
| Low | A `member_groups` row that disappears between validation and insert during publication membership replacement in `internal/exits/groups.go` hits `exit_groups_group_id_fkey` (`SQLSTATE 23503`) and returns HTTP `500 INTERNAL_ERROR` instead of `422 VALIDATION_FAILED`. | closed | #121 |

## What Held

- **`D-057` membership fencing and group creation (`internal/nodes/httpwire/review_d057_test.go`):**
  - `TestReviewD057_ConcurrentPublicationFencing`: Two concurrent `PUT /api/v1/admin/nodes/{id}/publication` calls with different `group_ids` and the same `If-Match` produced one `202` and one `412`/`409`, leaving `exit_groups` with exactly one of the two group sets.
  - `TestReviewD057_MembershipSurvivesUnpublishRevokeAndRepublish`: Omitting `group_ids` left existing `exit_groups` rows in place; revoking the node kept the row and `exit_count` at `1` while rejecting a later `PUT` with `publication_intent=published`; an explicit empty `group_ids: []` cleared the row (`exit_count: 0`).
  - `TestReviewD057_GroupCreateRaceAndNormalization`: Eight concurrent creates of the same `code` or shared `name` produced one `201` and seven `409`s via `UNIQUE(code)` and `UNIQUE(name)` in `internal/nodes/groups.go`. Uppercase/invalid `code` values were rejected (`422`/`400` on `^[a-z0-9][a-z0-9_-]{0,63}$`).
- **Member-facing responses never leak group identity (`§7`, `TestReviewD057_MemberResponsesDoNotCarryGroupIdentity`):**
  - `GET /api/v1/agent/desired-state` and `GET /api/v1/member/exits` returned `200` and contained neither the attached nor the unattached group's `id`, `code`, or `name`.
  - As a member, `GET /api/v1/admin/groups?code=...`, `GET /api/v1/admin/nodes/{id}`, and `POST /api/v1/admin/groups` all returned identical `403` responses before reaching the handler (`internal/httpapi/middleware.go`).
- **Resolver document fetch authorization and agent verification (`D-056`, `TestReviewD056_DocumentFetchAuthAndIdenticalNotFound`, `TestReviewD056_TruncatedAndSizeClaimDoNotApply`, `TestReviewD056_ForgedETagDoesNotHideTheNewDocument`):**
  - Unknown document IDs and unissued document IDs returned identical `404 NOT_FOUND` responses; `DELETE FROM dns_resolver_documents` was rejected by the `immutable record` trigger (`db/migrations/00180_dns_publish.sql`); revoked node certificates received `403 IDENTITY_REVOKED` and foreign-CA certificates received `401 UNAUTHENTICATED`.
  - On the agent (`node/agent/resolver.go`), a truncated body or a `SizeBytes` mismatch failed closed with `SIZE_MISMATCH` and `Apply` calls stayed `0`; a missing applier failed closed with `DEPENDENCY_UNAVAILABLE` (`TestReviewD061_MissingResolverStaysNotRun`); a forged `desiredETag` did not hide a new resolver document.
- **Resolver admin credentials (`D-061`, `TestReviewD061_CredentialAbsentFromLogsObservationAndAttestation`):**
  - `credentialMode` (`node/agent/dnsconfig.go`) required a regular `0600` root-owned file with no setuid/setgid/sticky bits; a `0644` credential file caused reload to fail closed (`dnsApply = nil`, static detail `"Resolver credentials are not usable."`) and leaked neither the secret nor the file path to structured logs, observations, or `health-attest.json`.

## Detailed Review Notes (Archived from `.agent/REVIEW.md`)

### [High] Published resolver documents are never placed in desired state
- **Where:** `internal/desiredstate/compiler.go:463-466`, `internal/nodes/httpwire/document_inputs.go:18` (`ResolveDocumentInputs`), `internal/nodes/resolver_documents.go:50` (`RecordIssuedResolverDocumentTx`).
- **Scenario:** When a DNS profile was published through the real publisher, `CompileTx` had no production caller wiring `ResolveDocumentInputs` or `RecordIssuedResolverDocumentTx`. The owning node's signed desired state had no `resolver_document` reference and `GET /api/v1/agent/resolver-documents/{id}` returned `404 NOT_FOUND`.
- **Proof:** `TestReviewD056_OwningNodeReceivesPublishedDocument` (`internal/nodes/httpwire/review_d056_test.go`).
- **Fix (#109, #121, #127):** Wired `WithDocumentInputs(resolve, issued)` into the production desired-state compiler transaction (`#109`), retained the review proof (`#121`), and added periodic resolver-document reconciliation (`min(reconcile_seconds, 60)s`) so newly eligible documents are compiled into signed desired state without requiring a manual publication command (`#127`).

### [Medium] Resolver admin HTTP client follows redirects
- **Where:** `node/dns/client.go:243-245` (`newHTTPClient`), `node/dns/client.go:219-222` (`Applier.call`).
- **Scenario:** `newHTTPClient` left `CheckRedirect` nil, so a `307` redirect from the configured host-local admin URL was followed (forwarding the Basic `Authorization` header on a same-host port change) and reported as a successful apply.
- **Proof:** `TestReviewD061_AdminClientRefusesOffOriginRedirect` (`node/dns/review_d061_test.go`).
- **Fix (#120):** Set `CheckRedirect` in `newHTTPClient` to refuse every redirect and return a fixed `HTTP <code> from <method> <path>` error without echoing any URL or credential.

### [Medium] A health answer can report `dns_policy` passed before the named document is applied
- **Where:** `node/agent/resolver.go:57-64`, `node/agent/decide.go:285-287`, `node/agent/attest.go:127-132, 214`, `node/agent/engine.go:418-429`.
- **Scenario:** When `e.revoked` was set or a lease-revoked branch called `answerHealth` before `evaluate`, `dnsDocument` was false and `Decide` attested `dns_policy: passed` ("policy in force") even though the named `resolver_document` had not been applied for that step/generation.
- **Proof:** `TestReviewD056_HealthAnswerBeforeDNSApply` (`node/agent/review_d056_test.go`).
- **Fix (#126):** Recorded the applied `(document_id, content_sha256)` in `e.appliedDNS` and failed `dns_policy` ("policy not in force") whenever the in-force desired state names a `resolver_document` that does not match `e.appliedDNS`.

### [Low] A group that disappears during publication replace is a 500
- **Where:** `internal/exits/groups.go:21-30`.
- **Scenario:** Counting `member_groups` without `FOR UPDATE` allowed a concurrent delete of the group row before `INSERT INTO exit_groups` to trigger `exit_groups_group_id_fkey` (`SQLSTATE 23503`), returning HTTP `500 INTERNAL_ERROR`.
- **Proof:** `TestReviewD057_UnknownGroupAndDeleteRace` (`internal/nodes/httpwire/review_d057_test.go`).
- **Fix (#121):** Mapped `SQLSTATE 23503` on `exit_groups_group_id_fkey` to `422 VALIDATION_FAILED` (`/group_ids` `unknown group`) with full transaction rollback.
