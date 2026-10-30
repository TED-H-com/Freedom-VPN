# Adversarial Review `r10-d063-review`: Synthetic-Probe Exception, Health Attestation, Updater Journal Snapshot, and Resolver Credentials (`D-063`, `D-064`, `D-061`)

- **Branch:** `r10-d063-review` (commit `42299c3`)
- **Date:** 2026-10-30 (`2026-10-30 01:25:04 -0400`)
- **Reviewing model:** `antigravity/gemini-4-argon` (recorded in commit `42299c3` `Agent:` trailer; not stated in `.agent/REVIEW.md` body)
- **Decision / scope reviewed:** `D-063` `§13.9` synthetic-probe exception in `node/enforce`, and the node agent's health attestation, observation, updater journal snapshot (`node/internal/updatejournal`), and `D-061` resolver-admin credentials in `node/agent` (`UPD02`–`UPD05`, `ISO01`, `MTR01`–`MTR05`, `DNS01`–`DNS04`, `SEC01`, `§9`, `§13.6`, `§13.9`)
- **Adopted proof files on `main`:**
  - [`node/enforce/review_d063_test.go`](../../node/enforce/review_d063_test.go) (adopted in `#144`)
  - [`node/agent/review_d063_test.go`](../../node/agent/review_d063_test.go) (adopted in `#150`, alongside [`node/agent/followup10_test.go`](../../node/agent/followup10_test.go))

## Findings

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | H-1: Post-probe `VerifyBaseline` failure in `node/agent/probeexc.go`, `node/agent/engine.go`, and `node/agent/attest.go` logs and discards cleanup errors without recomputing `e.decision`, attesting `internet_only: passed` and `tunnel_probe: passed` while `baselineOK` is `false`. | closed | #150 |
| High | H-2: `installProbeException` in `node/agent/probeexc.go`, `node/agent/engine.go`, and `node/agent/attest.go` reads stale `e.decision.Serve` from the previous step and re-applies member bindings while the `§13.9` `forwarding.blocked` latch is active. | closed | #150 |
| High | H-3: `VerifyBaseline` and `VerifyServing` in `node/enforce/parse_nft.go` and `node/enforce/verify.go` strip prefix lengths from CIDR strings in `elemFromParts`, silently skip `nft -j` prefix/range objects in `parseSetElems`, and ignore `"expires"` when `"timeout"` is non-zero in `@probe_v4` and `@probe_dst_v4`. | closed | #144 |
| High | H-4: `classifyDNS` and `noteDNSApply` in `node/agent/resolver.go` do not verify that the applier's `DocumentHash` or `PolicyGeneration` matches the signed `ResolverDocumentRef`, attesting `dns_policy: passed` on an applier digest or generation mismatch. | closed | #150 |
| Medium | M-1: `withProbeException` and `challengeProbe` in `node/agent/probeexc.go` install the probe exception before `ch.IssuedAt` and re-install it on every `Step()` after the challenge nonce was already answered, extending the kernel timeout past `t0 + ProbeMaxSeconds`. | closed | #150 |
| Medium | M-2: `Render` and `RenderBaseline` in `node/enforce/render_nft.go` and `node/enforce/validate.go` have no probe timeout ceiling (`TimeoutCeilSeconds` / `State.ProbeRemain`), allow `Destinations[i].AddrV4` to overlap `m.Bindings[i].AllowedV4`, and enforce no `ProbeMaxDestinations` cap. | closed | #144, #150 |
| Medium | M-3: `VerifyBaseline` and `VerifyServing` in `node/enforce/verify.go` return an untyped `fmt.Errorf` from `validateProbe(m)` instead of wrapping model-bound violations with `ReasonProbeUnbounded` (`PROBE_UNBOUNDED`). | closed | #144 |
| Medium | M-4: Serving probe installation in `applyProbeModel` (`node/agent/probeexc.go`) discards `ApplyResult.Resets` from `e.plane.Apply`, restoring `fvpn_generation` in the kernel before `e.enforce()` runs and swallowing external meter resets. | closed | #150 |
| Medium | M-5: `attachUpdate` in `node/agent/observe.go` unconditionally overwrites `rep.CurrentReleaseId` from journal `meta`, wiping it to `null` when `snap.CurrentRelease` is empty or reporting an unverified release ID before `locally_succeeded` or `rolled_back`. | closed | #140, #150 |
| Low | L-1: `credentialMode` in `node/agent/dnsconfig.go` checks mode `0600` and `Uid == 0` on resolver credential files but does not reject hard-linked files (`st.Nlink != 1`) or non-root group ownership (`st.Gid != 0`). | closed | #150 |

## What Held

- **`D-063` model validation and baseline member isolation (`TestReviewD063_Held_ProbeBoundsAndMemberIsolation` in `node/enforce/review_d063_test.go`, `TestRender_ProbeRejects` in `node/enforce/probe_test.go`):**
  - Non-`/32` `SourceV4` or `Destinations[i].AddrV4`, destinations in `SpecDeniedV4()` (`10.0.0.0/8`, `100.64.0.0/10`, `127.0.0.0/8`, `169.254.0.0/16`, `169.254.169.254/32`, `172.16.0.0/12`, `192.168.0.0/16`, documentation ranges) or `HostV4`, `Port == 0`, `Proto` outside `tcp|udp`, `ExpiresAt` in the past / `< 1s` / `> ProbeMaxSeconds`, and `PeerID` or `SourceV4` colliding with `m.Bindings` are rejected by `Render` and `RenderBaseline`.
  - In `RenderBaseline`, the `forward` chain has `policy drop` and every `accept` rule matches `@probe_v4`, so a real member packet cannot match `@probe_v4`.
  - `VerifyBaseline` rejects missing timeouts and `timeout > ProbeMaxSeconds` with `ReasonProbeUnbounded`, and rejects extra elements in `@probe_v4` with `ReasonIsolation`.
- **`D-063` diagnostic counter isolation from member usage (`TestReviewD063_Held_ProbeCountersIsolatedFromMemberUsage` in `node/enforce/review_d063_test.go`, `TestReviewD063_Held_AgentProbeCrashAndUsageCounters` in `node/agent/review_d063_test.go`):**
  - `probe_up` and `probe_down` use dedicated named counters, are filtered out by `diagnosticNamespace()` in `enforce.Samples()` and by `usageOnly()` in `node/agent/meterreset.go`, and never enter `agentstore.Counters`, `usagestore`, or `fakecontrol` usage uploads.
  - A simulated crash mid-probe leaves kernel set elements bounded by `timeout <= ProbeMaxSeconds`, and the next `Boot()`/`Step()` replaces the chains and deletes `probe_v4` and `probe_dst_v4`.
- **Health challenge before resolver document apply and stale ETag (`TestReviewD063_Held_HealthBeforeDNSApplyAndStaleETag` in `node/agent/review_d063_test.go`):**
  - When a named resolver document is only in `ResolverCache` (downloaded, not yet applied), or when desired state advances from `ref1` (applied) to `ref2` (unapplied), or when the fetched body fails SHA-256 verification (`HASH_MISMATCH`), `guardNamedDNS()` calls `refuseUnapplied(ref)` and `answerHealth` attests `dns_policy: "failed"`.
- **Updater journal snapshot hardening (`TestReviewD063_Held_JournalSnapshotHardening` in `node/agent/review_d063_test.go`):**
  - A 16 MiB garbage file, a truncated SQLite header, a FIFO (`syscall.Mkfifo` rejected by `os.Lstat` `!IsRegular()` without opening or blocking), a symlink to `/dev/null`, a database with `PRAGMA user_version = 99`, and a database locked by `BEGIN EXCLUSIVE` (bounded by `_pragma=busy_timeout(50)` and `SnapshotTimeout = 1s`) all finish within the deadline, produce `update.generation = null`, keep `CurrentReleaseId = e.cfg.ReleaseUUID()`, and leak neither the file path nor any journal bytes to logs or observations.
- **`D-061` adversarial admin URLs, symlinks, and SIGHUP with half-written file (`TestReviewD061_Held_ResolverAdminURLsAndSIGHUPHalfWrittenFile` in `node/agent/review_d063_test.go`):**
  - `http://127.0.0.1@evil.example/`, `http://[::ffff:127.0.0.1]:3000` (`ip.Is4In6()` rejected in `parseHostIP`), `http://localhost.`, and `http://127.1` are rejected by `validateDNS()` and `reloadResolvers()`.
  - A `0600` symlink inside a world-writable (`0777`) directory is rejected by `os.Lstat` before opening, and `reloadResolvers()` (SIGHUP) with a half-written JSON credential file fails closed (`e.dnsApply == nil`, `dns_policy: "failed"`) without leaking the partial credential secret or file path.

## Detailed Review Notes (Archived from `.agent/REVIEW.md`)

### [High] H-1: Post-probe `VerifyBaseline` failure still attests `internet_only: passed` and `tunnel_probe: passed`
- **Where:** `node/agent/probeexc.go:73-75`, `node/agent/engine.go:446-454`, `node/agent/attest.go:86-96, 131-156`.
- **Scenario:** When removing the probe exception (`clear()`) or re-applying the baseline in `e.enforce()` failed `VerifyBaseline`, `clear()` logged and ignored the error without clearing `e.baselineOK` or invalidating `e.lastProbe`, and `Step()` / `answerHealth()` used the stale `e.decision` computed before `e.enforce()`, attesting `internet_only: passed` and `tunnel_probe: passed` while `e.baselineOK == false`.
- **Proof:** `TestReviewD063_Agent_BaselineFailureAfterProbeStillAttestsPassed` (`node/agent/review_d063_test.go`).
- **Fix (#150):** Cleanup failure clears `e.baselineOK` and `e.serving`, marks the probe `probe_clear_failed` (never attested as `passed`), recomputes `e.decision` after `e.enforce()` whenever baseline or meter health changes, and recomputes checks from current inputs in `answerHealth`.

### [High] H-2: `installProbeException` uses stale `e.decision.Serve` from the previous step and re-applies member bindings while `forwarding.blocked` is latched
- **Where:** `node/agent/engine.go:402-447`, `node/agent/attest.go:71-81`, `node/agent/probeexc.go:41-61`.
- **Scenario:** When the `§13.9` `forwarding.blocked` latch engaged after a serving step, `evaluate()` ran `refreshProbe` before recomputing `e.decision = Decide(e.input())`. `installProbeException` saw the previous step's `e.decision.Serve == true` and called `e.plane.Apply` with the previous step's member `Bindings`.
- **Proof:** `TestReviewD063_Agent_ProbeUsesStaleServingDecisionWhenLatchBlocks` (`node/agent/review_d063_test.go`).
- **Fix (#150):** Refreshes the latch and computes a current decision before probing and again before installation and cleanup; losing a serving gate withdraws member forwarding to the baseline before any probe exception is applied.

### [High] H-3, [Medium] M-2, & [Medium] M-3: `node/enforce` probe set verification bypasses, timeout ceiling, destination cap, member collision, and `PROBE_UNBOUNDED` reason
- **Where:** `node/enforce/parse_nft.go:578-752`, `node/enforce/verify.go:25-27, 93-95, 254-303`, `node/enforce/render_nft.go:271-291`, `node/enforce/validate.go:193-281`.
- **Scenario:** `elemFromParts` stripped prefix lengths from CIDR strings without requiring `/32`; `parseSetElems` silently skipped `nft -j` prefix and port-range objects; `timeoutFromMap` ignored `"expires"` when `"timeout"` was non-zero; `validateProbe` lacked `ProbeMaxDestinations = 4`, member `AllowedV4` destination overlap checks, and `TimeoutCeilSeconds` / `State.ProbeRemain` re-render caps; `VerifyBaseline` returned an untyped error instead of `ReasonProbeUnbounded`.
- **Proof:** `TestReviewD063_VerifyBaseline_PrefixAndRangeElementsBypass`, `TestReviewD063_RenderBaseline_DestinationCollidesWithMemberAndUnboundedCount`, `TestReviewD063_OverlappingProbeResetsKernelTimeoutPastMax`, `TestReviewD063_VerifyBaseline_ModelBoundsReturnPlainErrorNotProbeUnbounded` (`node/enforce/review_d063_test.go`).
- **Fix (#144):** Required single-address `/32` CIDR elements, tainted sets on prefix/range/unknown element shapes (`PROBE_UNBOUNDED` on probe sets, `ISOLATION` on `allowed_v4`), checked `max(timeout, expires) <= ProbeMaxSeconds` (300) and `expires` against the model's remaining window plus 1s slack, enforced `ProbeMaxDestinations = 4` and non-overlap with member `AllowedV4`, capped re-renders via `ProbeException.TimeoutCeilSeconds` and `State.ProbeRemain`, and wrapped `validateProbe` errors in `ReasonProbeUnbounded`.

### [High] H-4, [Medium] M-1, M-4, M-5, [Low] L-1, & `D-064`: Agent DNS read-back verification, single-install challenge window without update lock, serving-probe meter resets, journal release provenance, and single-link credentials
- **Proofs:** `TestReviewD063_Agent_DNSPolicyPassesOnApplierDigestMismatch`, `TestReviewD063_Agent_ProbeInstalledOutsideWindowAndRepeatedEveryStep`, `TestReviewD063_Agent_ServingProbeSwallowsExternalMeterReset`, `TestReviewD063_Agent_JournalSnapshotOverwritesReleaseIDWithUnverifiedOrNullPointer`, `TestReviewD061_Agent_HardLinkedCredentialFileNotRejected` (`node/agent/review_d063_test.go`).
- **Fix (#140, #150):** Verified applier read-back `DocumentHash` (case-insensitive, trimmed) and decimal `PolicyGeneration` against the signed `ResolverDocumentRef` (`H-4`); authorized `installProbeException` from the live challenge without acquiring `/var/lib/vpn-node/update.lock` (`D-064`), requiring `issued_at < now < expires_at` with `>= 1s` remaining and window `<= 300s`, installing at most once per challenge nonce (`M-1`); fed serving-probe `ApplyResult.Resets` into `noteMeterReset` (`M-4`); kept `current_release_id` as the running binary's configured release unless a valid nonzero job ID reached `locally_succeeded` or `rolled_back` (`M-5`, alongside central observation authority in `#140`); and required mode `0600`, owner `0:0`, and `st.Nlink == 1` on resolver credential files (`L-1`).
