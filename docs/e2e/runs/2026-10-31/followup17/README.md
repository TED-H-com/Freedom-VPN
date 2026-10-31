# Follow-up 17 — installer wrappers, shared exports and real blockers

TEST ONLY, `e2e / 18000`. Latest complete run: `a7c41436-82b3-4665-b75c-e2887f2e1009`, source
`d37720e9062b3ff4528ec0213d38c10d1b6309f3`. No product source or service unit
was changed by this lane. The run has **8 passed, 1 blocked, 3 failed** and the
focused Go command exits **1**. This is not a full AT07 or upgrade pass.

## ENR-06 outcome

Initial installation exits 0 (`installed`), the real idempotent rerun exits 0
(`already_current`), and its quiescent before/after hashes match. The NetBird
socket `/run/fvpn-netbird/netbird.sock` exists and `/run/vpn-node` survives
consumer stops. Explicit restoration reads agent/watchdog/timer active and
updater oneshot inactive. First-poll capture shows absent desired state and the
fresh updater's no-work result.

Controlled reinstall follows operator revoke (202), preserves Node ID, creates
a new incarnation and retires identity/bound state with matching SHA-256
values and mode 0700. Both direct and installer-proxy requests with the retired
certificate return 403. Usage DB/WAL hashes are preserved; **this does not prove
nonzero member history or old-incarnation backlog delivery**. Every persisted
digest has a self-describing `*_sha256` field; missing files remain `absent`.

The actual journaled upgrade stages successfully and its apply process exits
0, but durable status is `failed`, stage `final_sample`, journal `rolled_back`,
health null, with the original current release unchanged. The genuine
`state/meter.checkpoint` is absent. No synthetic checkpoint is supplied; later
candidate restart/health remains unverified.

## Scenario table

| Scenario | Result | Requirement / evidence |
|---|---|---|
| [admin-login](a7c41436-82b3-4665-b75c-e2887f2e1009/admin-login.md) | pass | §14.2 |
| [serving-catalogue](a7c41436-82b3-4665-b75c-e2887f2e1009/serving-catalogue.md) | pass | §9, §14.2 |
| [installer-wrappers](a7c41436-82b3-4665-b75c-e2887f2e1009/installer-wrappers.md) | pass | §6.3, ENR-06, AT07 |
| [publish](a7c41436-82b3-4665-b75c-e2887f2e1009/publish.md) | blocked | §5.2, §14.2 |
| [fixture-member-admission](a7c41436-82b3-4665-b75c-e2887f2e1009/fixture-member-admission.md) | fail | §8, AT02, AT12 |
| [installer-usage](a7c41436-82b3-4665-b75c-e2887f2e1009/installer-usage.md) | fail | §10.5, ENR-05 |
| [capture-and-boot](a7c41436-82b3-4665-b75c-e2887f2e1009/capture-and-boot.md) | pass | AT23, ISO01, F9-06 |
| [installer-reenroll](a7c41436-82b3-4665-b75c-e2887f2e1009/installer-reenroll.md) | pass | ENR-06, AT07, §6.3 |
| [installer-journaled-upgrade](a7c41436-82b3-4665-b75c-e2887f2e1009/installer-journaled-upgrade.md) | fail | UPD05, AT26, §13.5 |
| [enroll-bootstrap](a7c41436-82b3-4665-b75c-e2887f2e1009/enroll-bootstrap.md) | pass | §6, §19 AT06 |
| [regular-kernel-boot](a7c41436-82b3-4665-b75c-e2887f2e1009/regular-kernel-boot.md) | pass | AT23, F9-06 |
| [node-restart](a7c41436-82b3-4665-b75c-e2887f2e1009/node-restart.md) | pass | §5, §6, F2-05 |

## Additional installer/enforcement blocker

The installer creates the empty non-hook `inet fvpn owner` chain. The real
agent's verifier requires exactly its four rendered chains, so the fifth
chain causes `HOOK_PRIORITY_MISMATCH`. Publication does not complete and the
counted member prerequisite fails. The same-slot held-usage/restored-traffic
row therefore fails without inventing counters or acknowledgements.
[Live chain and journal evidence](a7c41436-82b3-4665-b75c-e2887f2e1009/installer-owner-chain.md).
The harness does not remove the marker or weaken the verifier. The product
owners must reconcile `packaging/installer/isolation.go` and
`node/enforce/verify.go`; neither is owned by this lane.

## What the remaining passes establish

The real origin `tcpdump` produced a 204-byte capture exported to disk with a
SHA-256 receipt. A separate 3,145,728-byte export exceeded the diagnostic limit
and completed intact; diagnostic overflow returned an explicit error. The
private packet/receiver files are not committed. This proves capture plumbing,
not AT13's full continuous failure matrix.

Ordinary and installer agents use their kernel boot identity. The ordinary
enrolled agent heartbeat and updater lock agree with the kernel. A real
container restart retains the kernel ID and reaches a fresh accepted heartbeat
and observation. This is process-restart evidence, not OS reboot evidence.

The installed fixtures use real amd64/arm64 archives and public registration
with worker verification. arm64 execution on hardware remains unmeasured.
[Exports and lifecycle](../../../installer-layout.md),
[AT29 operator guest procedure](../../../at29-guest-clock.md), and
[dated lane responses / portal audit](../../../RESPONSES.md).

## Preserved attempts

These directories retain their original observations. Later passes do not
rewrite failed rows or convert an interrupted run into acceptance evidence.

| Attempt | Source | Outcome / reason |
|---|---|---|
| [5119af0b](5119af0b-9fb7-4257-b32a-a6595ee1aa03/admin-login.md) | `7af2b39` | Interrupted with SIGTERM after shared-host admission changed; 2 setup passes only, no completion claim. |
| [a3605492](a3605492-0c78-4bf1-8219-4108269a0788/installer-wrappers.md) | `7af2b39` | Exit 1, 2 pass / 1 fail. Holding the whole agent prevented bootstrap/setup-key delivery. The shared barrier now withholds only authenticated desired-state GETs. |
| [a20daa7b](a20daa7b-fa3f-4980-946a-3edbde3f0ca7/installer-wrappers.md) | `32a81e3` | Exit 1 after 3 passes. Serving setup tried a host read of the root-owned resolver credential file. The fix pipes the private bytes between owned containers. |
| [f1a17b4e](f1a17b4e-c671-4186-9a6e-b5b7f0033d37/installer-journaled-upgrade.md) | `6e3667f` | Exit 1, 6 pass / 1 blocked / 5 fail. Late AdGuard startup collided with NetBird's DNS stub; diagnostic overflow was bypassed by promoted `bytes.Buffer.ReadFrom`; the normal restart assertion retained the installer's incarnation. All three harness issues were fixed. Its genuine missing-checkpoint failure remains. |
| [a7c41436](a7c41436-82b3-4665-b75c-e2887f2e1009/installer-wrappers.md) | `d37720e` | Exit 1, 8 pass / 1 blocked / 3 fail. Capture and restart now pass; installer chain mismatch blocks serving, and genuine checkpoint remains absent. |

The f1a17b4e runner was compiled from its recorded source; later fixture helper
builds could observe subsequent harness commits made during diagnosis. The
latest complete run kept runtime source fixed throughout.

## Reproduce

Follow shared-host admission limits, then run in the foreground:

```sh
FVPN_E2E_PREFIX=e2e FVPN_E2E_PORT_BASE=18000 \
  make -f mk/e2e.mk e2e-up E2E_RESET=1
FVPN_E2E_PREFIX=e2e FVPN_E2E_PORT_BASE=18000 FVPN_E2E_FOLLOWUP17=1 \
  go test -tags e2e -count=1 -v -timeout 30m ./tests/e2e \
  -run '^TestE2E_Followup17Installer$'
FVPN_E2E_PREFIX=e2e FVPN_E2E_PORT_BASE=18000 \
  make -f mk/e2e.mk e2e-down
```

Always tear down even when the required live check fails. Use task disk scratch
under `.agent/tmp` for `TMPDIR` and `GOTMPDIR`. Do not replace durable failures
with process exit codes or change host networking/time to make this pass.
