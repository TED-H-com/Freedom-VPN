# r11-d064-review

Adversarial review of the D-064 probe exception, the D-063 issuer, zero-binding publication, and accepted-result authority. Branch `r11-d064-review`. Proofs are the `TestReviewR11_*` tests. Failing tests are marked `TODO(review)` and expect the safe property. `go vet -tags integration` is clean for `node/agent`, `node/updater`, `node/internal/controlclient`, `internal/nodes`, `internal/nodes/httpwire`, and `internal/fleetupdate`. No product code was changed.

## High

### Probe cleanup puts member forwarding back while the updater holds the lock

Where: `node/agent/probeexc.go:33` (`installProbeException`), `node/agent/probeexc.go:69` and `:83` (`applyProbeModel` on install and on cleanup), `node/agent/engine.go:684` (`enforce` still takes the lock).

Scenario: Generation 2 is applied and serving with one member binding. A test process holds `nodelock.TryAcquire` on the agent lock path and installs the updater baseline (generation 0, zero bindings). A live challenge then arrives. The challenge file was written by this test process, which is the same euid as the agent and is not the updater. Install and cleanup both `Apply` the captured serving model. Install puts generation 2 back with the member binding and the probe `100.64.8.1/32`. Cleanup removes the probe and applies generation 2 again with the member still bound (`peer-a1`). A following `enforce` under the same lock does not apply and records `UPDATE_BUSY`.

Proof: `TestReviewR11_ServingProbeIgnoresUpdateLock`

```
review_r11_d064_test.go:102: while the update lock was held, probe install applied generation 2 with 1 member bindings and probe 100.64.8.1/32, then cleanup applied the same generation with probe removed (bindings 1, peer peer-a1). updater baseline bindings were 0. enforce under the same lock did not apply (applies stayed 4, UPDATE_BUSY=true). challenge nonce 7d66b20ce066d5e9b3ed9dcb08e30250fdfe295ee40cce79f669ba4ed8724e37 was written by this test process, not the updater
```

Fix: agent. Install and removal of the exception have to leave the updater's baseline in place for as long as the update lock is held. The serving `Apply` in `enforce` already refuses that lock.

## Medium

### Any same-euid writer can open the probe hole the challenge file authorizes

Where: `node/internal/healthattest/io.go:142` (`checkDir`) and `:161` (`checkOwner`); `node/agent/attest.go:173` (`matchingChallenge`); `node/agent/probeexc.go:169` (`challengeProbe`) and `:221` (`probePeerOK`).

Scenario: The agent trusts `health-challenge.json` when the journal directory is a real directory with no group or other permission bits, the file is a regular file mode `0600`, and the owner uid equals the agent euid. There is no signature and no updater pid. "Names this process" is `NodeID`, `IncarnationID`, `BootID`, and `ReleaseID`. `TxnID` is carried through and is not tied to an updater transaction. `probePeerOK` checks that the peer id is non-empty, at most 128 bytes, and free of CR/LF. It does not know member peer ids.

A process with this euid wrote a challenge whose source is `10.9.8.7/32`, with four destinations (`1.1.1.1/32`, `8.8.8.8/32`, `192.88.99.1/32`, `192.31.196.1/32`) and `resolver_dns`. `installProbeException` installed it. The rendered rules accept those four destinations, and the DNS accept to resolver `10.0.0.53` is emitted before the `@denied_v4` drop (`node/enforce/render_nft.go:330`). The rendered ceil on this issuance was 42s. `TestReviewR11_IssuanceWindowAndRenderCap` shows an honest window of `expires_at = issued_at + 300s` is accepted and the single render stays within `ProbeMaxSeconds`. The shape of the hole is one `/32` source, four destinations, resolver DNS, and a kernel timeout of the remaining whole seconds up to that 300s window. Extending past the window is the separate restart finding below.

A member `/32` and a member peer id are installed when that member is absent from the current desired document. The node's SNAT `192.0.2.37/32` is refused. A destination of `169.254.169.254/32` is link-local, so `unusableV4` (`probeexc.go:225`) refuses it before install, and the issuer refuses it too. A node-local destination on a non-admin port passes `challengeProbe` and the dry-render then refuses it. An address contained in a current member `AllowedIpv4` is refused by `memberCollision` (`probeexc.go:246`). When the probe peer id equals a member who is in the current document, install is refused. When that member is absent, the same id is accepted.

Proof: `TestReviewR11_SameUIDChallengeOpensProbeHole`

```
review_r11_d064_test.go:180: same-euid writer (uid 1000, mode 0600, txn 01a18bc6-b68c-723c-8613-f0d739e86472, not the updater) installed source 10.9.8.7/32 dests 4 resolver_dns true ceil 42s. rendered accept for 192.88.99.1, 1.1.1.1, 8.8.8.8 and 192.31.196.1, and DNS accept to 10.0.0.53 precedes the deny drop (dns 570 deny 932)
```

Fix: agent. The challenge the agent installs needs an issuer the agent can distinguish from every other process of this uid, and the agent-side accept list has to cover the source ranges and the DNS-before-deny case the render currently opens.

### Issuer accepts special-purpose destinations and private sources, and has no node, control, or resolver address

Where: `node/enforce/prefixes.go:13` (`SpecDeniedV4`); `node/updater/probe.go:16` (`validateProbePeer`) and `:45` (`probeHostV4`).

Scenario: Of the ranges named for this review, `SpecDeniedV4` already contains `0.0.0.0/8`, `100.64.0.0/10`, `192.0.0.0/24`, `198.18.0.0/15`, `240.0.0.0/4`, `255.255.255.255/32`, and the three documentation ranges `192.0.2.0/24`, `198.51.100.0/24`, and `203.0.113.0/24`. It does not contain `192.88.99.0/24`. `validateProbePeer` also accepts destinations in `192.31.196.0/24`, `192.52.193.0/24`, and `192.175.48.0/24`. Destinations inside `SpecDeniedV4`, including `169.254.169.254/32`, are refused.

`probeHostV4` requires `IsGlobalUnicast`, a first octet other than 0, and a first octet below 224. In this toolchain `IsGlobalUnicast` is true for RFC1918, so the issuer accepts sources `10.9.8.7/32`, `172.16.5.5/32`, `192.168.1.9/32`, `100.64.8.1/32`, and `192.0.2.8/32`.

`validateProbePeer` takes only the probe object. It has no input for the node's public address, the control URL, or the configured resolvers. The stand-ins `9.9.9.9/32`, `1.1.1.1/32`, and `8.8.8.8/32` are accepted. A public destination is otherwise a legitimate probe target; the gap is the missing comparison with those three configured addresses.

Proof: `TestReviewR11_IssuerSpecialPurposeDestinations`

```
review_r11_d063_test.go:71: SpecDeniedV4 is missing [192.88.99.0/24]. validateProbePeer accepted destinations [192.88.99.1/32 192.31.196.1/32 192.52.193.1/32 192.175.48.1/32 1.1.1.1/32 8.8.8.8/32 9.9.9.9/32] (1.1.1.1 stands in for a control address, 8.8.8.8 for a configured resolver, 9.9.9.9 for a node public address that is not in the updater config) and sources [10.9.8.7/32 172.16.5.5/32 192.168.1.9/32 100.64.8.1/32 192.0.2.8/32]
```

Fix: updater. Add the missing special-purpose ranges, reject private, CGNAT, and documentation sources, and refuse a destination that is the node's public address, the control URL's address, or a configured resolver.

### trustedNow installs a probe that wall time will not answer

Where: `node/agent/probeexc.go:41` (`installProbeException` uses `trustedNow`); `node/agent/attest.go:107` (`answerHealth` uses `clock.Wall`).

Scenario: The anchor is 30s ahead of the wall clock, inside the forward-skew limit, so `trustedNow` stays on the anchor. The challenge is issued at wall time with a 45s TTL. `trustedNow` is inside `(issued_at, expires_at)` and the exception is installed. Wall time equals `issued_at`, and `answerHealth` requires `now.After(issued_at)`, so no attestation file is written. The probe runs between install and the deferred clear, and the updater has no answer to accept.

The opposite skew is held: `TestReviewR11_WallAnswersDoNotSkipProbeInstall` keeps `trustedNow` before `issued_at` while wall time is inside the window. That step neither installs nor writes an attestation.

Proof: `TestReviewR11_TrustedNowInstallsWhenWallRefusesAnswer`

```
review_r11_d064_test.go:285: trustedNow 2026-10-30 07:47:28.331869456 +0000 UTC installed the probe for challenge issued 2026-10-30 07:46:58.331869456 +0000 UTC expires 2026-10-30 07:47:43.331869456 +0000 UTC, and wall 2026-10-30 07:46:58.331869456 +0000 UTC wrote no attestation
```

Fix: agent. Use one clock for the install decision and the answer decision.

### A new process reinstalls the same challenge, and ProbeRemain 0 extends it past the issuance

Where: `node/agent/probeexc.go:38` and `:66` (one installation per issuance is remembered only in this process); `node/enforce/render_nft.go:365` (`probeSeconds`). The agent supplies `TimeoutCeilSeconds`, so the hold branch at `:386` (which requires a zero ceil) does not apply to an agent install.

Scenario: The same challenge is legal in a second process because `probeChallenge` was cleared. With the sets already live and `ProbeRemain` 120, the second render is capped at `timeout 120s`. That cap held. After the wall clock steps back 80s, `trustedNow` falls back to the wall. The ceil computed from that earlier instant is 200s. `ProbeRemain` 0 makes the render flush `probe_v4` and add the element again with `timeout 200s`. The kernel clock is the advanced time, so the new element ends 80s after `expires_at` and 80s after `issued_at + 300s`.

Proof: `TestReviewR11_RestartRemainCapAndAnomalyExtension`. The honest `ProbeRemain` cap passed earlier in the same test. The failure is the re-add:

```
review_r11_d064_test.go:530: restart with ProbeRemain 0 re-added timeout 200s at kernel time 2026-10-30T07:48:18.398299017Z, ending 2026-10-30T07:51:38.398299017Z, which is after expires_at 2026-10-30T07:50:18.398299017Z and after issued_at+300s 2026-10-30T07:50:18.398299017Z
```

Fix: agent. A second install of one issuance has to cap the kernel timeout at the original `expires_at` even when `ProbeRemain` is 0 and `trustedNow` has fallen back to the wall.

### Zero-binding publication leaves the NetBird route up when the first binding arrives

Where: `internal/exits/exits.go:766` (`PublishExit` with `MemberGroupIds` runs before the data-path gate); `internal/exits/exits.go:826` (`PublicationDataPathFailure`); `internal/exits/exits.go:1232` (`failPublishStep` records the step and sets `publication_observed` to `error`, and does not withdraw the route). The exception predicate is `internal/nodes/publication_evidence.go:34`, which matches `signed.device_bindings` equal to `[]` on the current incarnation's envelope. `member_bindings`, groups, and plan `exit_group_ids` are absent from that SQL.

Scenario: An empty published generation acknowledges, the gate returns success, and the worker attaches one NetBird route for the publication group (`publication_intent` `published`, `publication_observed` `active`, withdraws 0). The compiler then issues generation 4 with one device binding. An observation that is still on the empty generation is rejected with `stale_generation` (that assertion passed before the failure below). An observation that acknowledges generation 4, with lease, QoS, and source-binding still `not_run`, is `serving_checks_unconfirmed`. The route count stays 1 and withdraws stay 0. The member peer is on the signed envelope, and the group route from the zero-binding publish is still attached, before lease, QoS, and source-binding have been verified for the generation that carries the binding.

`device_bindings` is the only envelope field the exception reads. The route admission that already happened is the node's `MemberGroupIds`, which the predicate does not see.

A previous incarnation's envelope at the same generation was not given its own database. The join at `publication_evidence.go:35` is `e.node_id = n.id AND e.incarnation_id = n.current_incarnation_id`. `desired_state_envelopes` is unique on `(node_id, incarnation_id)`, so a previous incarnation's row is not the row this predicate reads.

Proof: `TestReviewR11_ZeroBindingRouteBeforeFirstBindingChecks`

```
review_r11_publication_integration_test.go:107: generation 4 carries peer "TEST ONLY member peer" while lease/qos/source_binding are not verified (gate serving_checks_unconfirmed). NetBird route count 1, withdraws 0, publication_intent published, publication_observed active. The zero-binding publish attached the route for group 01a18bc6-bad5-7e0d-9a4b-de9d32bc0785 and nothing withdrew it
```

Fix: wire-nodes. Re-gate the NetBird route for the generation that first carries a binding, and withdraw it until `lease`, `qos`, and `source_binding` pass for that generation.

### Heartbeats move the serving release to a release that was never a job target

Where: `internal/fleetupdate/serving.go:98` (`observeTx`). Before any accepted `update_results` row for the incarnation, the heartbeat writes `nodes.current_release_id` and `nodes.last_good_release_id` from the observation (`serving.go:118`) and forces agreement. There is no predicate that the release is a target of this node's job.

Scenario: A node has a job for release `l.new` and has submitted no result. A second registered release, `1.3.0-review`, is not a job target (`update_jobs` count 0 for that pair). Heartbeat 2 sets both node release columns to that other release. Heartbeat 3 moves `current_release_id` to `l.old`. The accepted-result count stays 0.

Proof: `TestReviewR11_HeartbeatSeedsAnyReleaseUntilAResult`

```
review_r11_d064_integration_test.go:62: with no accepted result, heartbeat 2 set current_release_id and last_good_release_id to 01a18bc6-baa5-7640-8937-50ffc026f132, which is not job target 01a18bc6-ba9e-7929-baa8-a5659e78dc09, and heartbeat 3 moved current_release_id to 01a18bc6-ba97-74e5-aac0-462c485942d2
```

Fix: fleet-update. Seed the serving release from a heartbeat only when that release is a target of the node's job, and stop moving it once the pre-result window is no longer the authority you want.

## Low

### RenewLease returns inside Retry-After, and the agent tries again on the next call

Where: `node/internal/controlclient/client.go:341` (`statusFrom` copies `Retry-After` onto `StatusError`); `node/internal/controlclient/api.go:117` (`RenewLease`) and `:193` (`postJSON` returns that error immediately); `node/agent/sync.go:114` (`syncLease` logs the error and returns nil). `renewAppliedLease` (`sync.go:144`) calls `syncLease` again when the stored lease generation is still behind `appliedGen`.

Scenario: A 429 with `Retry-After: 30` comes back as `*StatusError` with `RetryAfter` 30s. `RenewLease` returns in 485µs. The agent does not read `RetryAfter`. On a sticky lease for an older generation, six steps issued 12 renewals in 6.6ms, two per step, with no delay. The default step interval is 10s (`config.go:125`), so a steady `Run` is two requests per 10s. A tight step loop, and the pair of calls inside one step, do not wait out `Retry-After`. Forwarding stays withdrawn with `policy_missing` (see the held lease section). The class limiter itself is out of scope.

Proof: `TestReviewR11_RenewLeaseReturnsBeforeRetryAfter`

```
review_r11_lease_test.go:42: Retry-After was parsed (30s) but RenewLease returned in 485.132µs
```

`TestReviewR11_StaleGenerationLeasePressure` passed and logged `renewals 12 in 6.617625ms (6 steps); serving stayed off with policy_missing`.

Fix: agent. On 429, wait the `Retry-After` carried on `StatusError` before the next `RenewLease`, including the second call in the same step.

## What I tried that held

Challenge identity and `Same`. `Challenge.Same` (`node/internal/healthattest/io.go:75`) compares `TxnID`, `Nonce`, `ReleaseID`, `NodeID`, `IncarnationID`, `BootID`, `IssuedAt`, `ExpiresAt`, `Components`, `Consumed`, and the probe (`PeerID`, `SourceV4`, `ResolverDNS`, each destination). `TestReviewR11_ChallengeSameFields` flips each of those and `Same` returns false. `TestReviewR11_RewriteDuringProbeDoesNotAttach` rewrites the file during the probe, keeping the nonce, and changing the expiry, the release, or the destination. No attestation is written, so the old probe result is not attached to the replacement.

Clocks and the 300s window. The wall-inside / trusted-before skew neither installs nor answers (`TestReviewR11_WallAnswersDoNotSkipProbeInstall`). `expires_at = issued_at + 300s` is accepted and the rendered timeout is the ceil, with no `timeout 301s`. `issued_at = now` is refused. `healthattest.Issue` accepts TTL 300s and refuses 300s plus one nanosecond (`node/internal/healthattest/io.go:42`, `TestReviewR11_IssuanceWindowAndRenderCap`). An honest second render with `ProbeRemain` 120 is capped at 120s inside the restart test, before the anomaly failure.

Collision bounds. `TestReviewR11_ProbePeerCollidesOnlyWithCurrentDocument`: a member id and `100.64.1.1/32` are accepted only while they are absent from the current document; both are refused once that document contains them; a probe id equal to a current member is refused; `probePeerOK` accepts `peer-a1`; the node SNAT source is refused; a node-local destination on port 9 passes the pre-check and the dry-render refuses it.

Issuer config. `TestReviewR11_MalformedProbePeerHidesTheValue`: a malformed `probe_peer` fails as `updater: invalid probe_peer` from `loadProbeConfig` and from `Engine`, and the error text does not contain the address. `TestReviewR11_ProbePeerIsASnapshot`: `cloneProbePeer` does not alias the destination slice (`host.go:171` stores the clone). A later `healthattest.Issue` mints a new nonce, so `Same` is false, and the 300s TTL is unchanged. SIGHUP calls `reloadResolvers` only (`node/agent/engine.go:364`). The updater has no reload of `ProbePeer` between `issueChallenge` (`node/updater/attest.go:317`) and the answer; a new issuance is a new nonce.

Cleanup and the next step. `TestReviewR11_ClearFailureNextStep` passed. After `probe_clear_failed`, the failed step's attestation is `tunnel_probe` failed / `probe_clear_failed`, `baselineOK` is restored by the same step's `baseline()` (`engine.go:641`), and that step records a baseline call and no serving apply. `!BaselineOK` adds `safety_check_failed` (`decide.go:132`), so `enforce` takes the `!Serve` path and calls `Baseline` with a nil probe. `RenderBaseline` on that datapath, with `probe_v4` and `probe_dst_v4` still in `State`, emits `delete set inet fvpn probe_v4` and `delete set inet fvpn probe_dst_v4` (`render_nft.go:407`). The next step does not install the same challenge again. Its attestation is `tunnel_probe` passed / `overlay_up`, and `lastProbe` is no longer `probe_clear_failed`. The test plane does not program a kernel; the delete lines are the production script `ApplyBaseline` renders after `loadState`.

Publication gate. `TestReviewR11_PublicationReasonVocabulary` passed. The eight step reasons in `docs/nodes/WIRING.md` (`forwarding_disabled`, `stale_generation`, `tunnel_not_passed`, `check_failed`, `serving_checks_unconfirmed`, `isolation_unconfirmed`, `readiness_unconfirmed`, `lease_unconfirmed`) are exactly the strings `publicationForwardingFailure` returns. Empty bindings with those three checks `not_run` return an empty cause. The same checks `not_run` with a binding return `serving_checks_unconfirmed`. These strings are the step `reason` vocabulary in that doc. They are not an OpenAPI enum. The integration test's stale-ack assertion passed: acknowledged generation G with `device_bindings=[]`, desired generation G+1 with one binding, observation still at G, returns `stale_generation`. The call passes G+1 as `minimumGeneration`, and applied G is below that minimum, so this proof takes the early return at `publication_evidence.go:25`. The same cause is returned at `:44` when the acknowledged generation is at least the caller's minimum and is still below `nodes.desired_config_generation`.

Accepted results. `TestReviewR11_HeartbeatConfirmationFlapsAroundAcceptedRelease` passed. After one accepted `succeeded` result, a disagreeing heartbeat, an agreeing heartbeat, and another disagreeing heartbeat set the target `Unconfirmed` flag true, then false, then true. `current_release_id` and `last_good_release_id` stay on the accepted release. The operator-facing flag is `unconfirmedSQL` (`internal/fleetupdate/progress.go:273`): a disagreement clears `update_jobs.last_observed_at` (`serving.go:131`), and `last_observed_at IS DISTINCT FROM` the fleet observation time marks the target unconfirmed. The release columns do not move.

`TestReviewR11_ParallelConflictIsOneRow` passed. Two identical conflicting submissions in parallel produce one `update_result_conflicts` row and one `update_result.conflict` audit. The insert is `ON CONFLICT DO NOTHING`, and the audit runs only when `RowsAffected` is non-zero (`results.go:116`).

`TestReviewR11_SubmitFailedStillRequiresErrorCode` passed. `failed` without `error_code` is rejected at stages `download`, `check`, and `report` (`results.go:173`). No job row is stored as `failed` with a null `error_code`. `TestReviewR11_FailedObservationDoesNotSkipErrorCode` passed. An observation with `Update.State=failed` and no error code does not move `update_jobs.state` to `failed`. There is no `result_receipts.go`. The observation path validates a present stage or state enum in `ObserveReportTx` and does not project a failed job by itself.

`TestReviewR11_RolledBackCandidateHealthOmittedAndNull` passed. A `rolled_back` result with `candidate_health` omitted and one with explicit null are both accepted. On the target, `CandidateHealth.Get` fails either way: `setTargetResultEvidence` uses `presence.FromPointer` (`progress.go:257`), and both JSON forms unmarshal to a nil pointer. The raw `observed_evidence` still differs. The omitted receipt has no `candidate_health` key. The explicit-null receipt contains `"candidate_health": null`.

Lease pressure and `lease_expired`. `TestReviewR11_StaleGenerationLeasePressure` passed. A sticky lease that stays on generation 1 after the desired generation moves to 2 clears serving. The heartbeat reason is `policy_missing`. The observation's forwarding reasons do not contain `lease_expired`. The lease check detail does not contain `lease_expired`, and its error code is not `LEASE_EXPIRED`. `decide.go:165` uses `policy_missing` for a lease of another generation and reserves `lease_expired` for the anchored deadline of the lease for the applied generation (`decide.go:268` reports `STALE_GENERATION` on that check when serving is being evaluated). The agent keeps the old lease in memory and does not serve on it.
