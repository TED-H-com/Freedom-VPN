# Follow-up 7 verification — 2026-10-30

Three complete fresh-stack runs used prefix `e2e`, base `18000`, on main through
`2a21468` (#102, including #98/#99/#96/#100/#101) plus this task's changes.
Each run has 21 scenarios. All finish **16 pass, 5 blocked, 0 fail**; none
is a green full acceptance suite. Run 3 replaces the entire dated record set.
Every final record has the same run ID and source commit. Runs 1 and 2 remain
in ignored task state; all three runs' manifest digests are retained below.
Run 2 adds stateless import wrappers, a manifest-option test and a boot-config
mode observation. Run 3 adds readback polling after the good feed is restored,
proving profile eligibility before the node-delivery check, and clarifies the
usage scenario title. The forwarding/resolver/helper behavior is unchanged.

| Run | Run ID | Source | Pass / blocked / fail | Test seconds | Go / make exit |
|---|---|---|---|---:|---|
| 1 | `13b67432-d37b-41e6-93b4-471ffce1f61a` | `f1729c7` | 16 / 5 / 0 | 292.247 | 1 / 2 |
| 2 | `7d5b5383-9c4a-4d89-a4f0-2fa6b0e18c8a` | `78fd428` | 16 / 5 / 0 | 311.944 | 1 / 2 |
| 3 | `dc2f6353-89ec-434c-bc68-567066d9fe51` | `b15ff05` | 16 / 5 / 0 | 310.316 | 1 / 2 |

All three runs pass group/plan activation, the first live paid-period reconciliation,
publication (`active`, all operation steps succeeded), real signed boot/candidates,
native exact-subject admission, and the 2 × 2 scale smoke. Compared with follow-up
6, catalogue, billing and publish move to pass; the scale smoke adds one scenario.

The remaining five blocked rows reduce to two product gates:

- **w2-agent-core:** desired/applied generation `5/5` contains an authorized
  device binding and rate policy. The latest stored lease is generation
  `1`, has `0` members and is the only lease. Its interval is
  `2026-10-30T04:17:40.551404+00:00` to `2026-10-31T04:17:40.551404+00:00`. Node detail returns 200;
  the accepted observation reports `LEASE_EXPIRED`, and heartbeat has
  `serving 0`, `policy_expires_mono_ns 0`, `reason lease_expired`.
  `needRenew()` considers the node deadline, not desired/member changes.
  No counter increment, native traffic, sample arrival or expiry success is claimed.
  The control-stop/deadline fault is not injected without a positive counted flow.
- **w3-wire-nodes / internal/desiredstate:** an eligible published resolver document
  exists, but signed desired state has no `resolver_document` reference. The real
  node mTLS GET returns **404 NOT_FOUND**; node detail is 200 with
  `dns_policy_evidence: null`. D-061 admin readback is 200, zero rules, query logs
  and statistics disabled, and the synthetic name remains unblocked. Bad-pin
  node last-good testing is blocked by that missing positive path; the lab does
  not fabricate a signed reference or projector row.

| Scenario (2026-10-30) | Result | Owner of blocked gate |
|---|---|---|
| `admin-login` | pass | — |
| `backup` | pass | — |
| `billing` | pass | — |
| `control-metrics` | pass | — |
| `desired-state` | pass | — |
| `enroll-bootstrap` | pass | — |
| `forwarding` | blocked | w2-agent-core: no current member lease / counted flow |
| `lease-expiry` | blocked | w2-agent-core: positive counted-flow prerequisite |
| `member-traffic` | blocked | w2-agent-core: refresh the member lease after desired changes |
| `native-scale-hook` | pass | — |
| `portal-billing-return` | pass | — |
| `publish` | pass | — |
| `release-register` | pass | — |
| `revoke` | pass | — |
| `serving-catalogue` | pass | — |
| `stack-health` | pass | — |
| `threat-feed-central` | pass | — |
| `threat-feed-node` | blocked | w3-wire-nodes / internal/desiredstate: issue resolver reference |
| `update-bad` | pass | — |
| `update-good` | pass | — |
| `usage` | blocked | w2-agent-core prerequisite; w3-wire-usage follows positive flow |

## Release digests

Boot 0.1.0, good 1.0.1 and tamper target 1.0.2 contain real tree-built agent,
watchdog and updater binaries. NetBird's manifest component is exactly 0.82.0.
Artifacts are distinct. The bad scenario flips real archive bytes after release
verification; no fixture binary substitutes for a candidate. Manifest rollback
`allowed_release_ids` defaults to `[]` (D-034e); restricted callers can include
last-good through the exported helper. Serving support is a separate API policy.

| Run | Version | Manifest envelope SHA-256 | Signed object SHA-256 |
|---|---|---|---|
| 1 | 0.1.0 | `56b4dda6a740a715990fa6b9bf8faf0bda18fc290093791218a49489101013b2` | `69b279f7c790ae311ebc276e1b75a48f9467519c37103fb668e82111089785bd` |
| 1 | 1.0.1 | `d2e7ce442b91cdb5ec5f13ff672d4de8a533538708e2f7d7f1ac762cb5a88e70` | `f6d0de4e9694b30b0d175aef3a921c9e1cf51927b355de507e9cbce87ebea9cb` |
| 1 | 1.0.2 | `0636f28f19ed52f5a61b6dc1024ab416bc8d0a6440c90cfec9f6ce1e761e6f21` | `964efec9cb9a1174e731b3a9be10684b502bc4abcc6ce15af6fadfae39547435` |
| 2 | 0.1.0 | `f23c7f0e223b24a7dbba3cca44ab1741f8bd205c2b42a8a0d47f7c2d9848d3dd` | `b5f4c31c08d23c1ae20ef7379a5764e6e7eb31cadf64f8002ab237ecbe11f036` |
| 2 | 1.0.1 | `014fb724c9e3ec3a764816f8f4bda219a5d8e5d694c75e9fe69dd49e0630f674` | `3cc272f8314d41ed246734a4488299bcad09800e30cd368580fcee14dcd76edf` |
| 2 | 1.0.2 | `8cb9537475b5e85253a3e22a44ce036864ae6820daed381875a128a4e620d571` | `8645e18599d0d94cd921bf5bf85e74bdbb20cb8d8688116e33571d696c7bc6ce` |
| 3 | 0.1.0 | `66807d1cff4f45496058c45f6e4d0756e2352fb6176bda1733c12ce7618aa27d` | `c2f1435123abb579bad8bc8899d1e71abcf2e933e565f75c3d2ac0b588ee8367` |
| 3 | 1.0.1 | `d009cb7718d0e1b72c6f4a73eb5aba6d8ee6a9158f206511f120af76a1571845` | `e6b3797da9e9b004611855fdc7688812e2933e8e52eaaf3b305c2a723a437d32` |
| 3 | 1.0.2 | `9c9ea639b768ac55f950a839a469c788e3a0afb702d38f2a8b59cb2a26b8dd54` | `c0af8189fb076129cbf63ce4b84b8014113652da4e7d10c9fab78cd02e49ded5` |

| Version | Architecture | Run 3 artifact SHA-256 |
|---|---|---|
| 0.1.0 | amd64 | `d15ef01352f4db345127db122968305e1284f0c79343e3e6c730238498de253e` |
| 0.1.0 | arm64 | `a50819dd1e0e908b7f61e71e541a39e728cc3b68638a9122642b040656e7ed31` |
| 1.0.1 | amd64 | `deac73e443c072229aaae81028158048c1cb24b94e69e05a72d9155ce27106da` |
| 1.0.1 | arm64 | `f385446fe4c4930a6de871ea1d5e611056519cabca76213e33149fba3612fc3e` |
| 1.0.2 | amd64 | `a6f23ea7cdd0df58176b9afc0ff07b9cb45f2217235b1eafb5a83d5026001e50` |
| 1.0.2 | arm64 | `6ab419111cd520db1cd21d2ae3da96ce564e3d6bf102aa2ff0fe891a45c4200f` |

## Self-check and cleanup

`make -f mk/e2e.mk e2e-lint`, `make go-lint`,
`go vet -tags e2e ./tests/...`, and
`go test -count=3 ./tests/e2e/... ./tests/e2e/cmd/... ./tests/e2e/internal/...`
all exited 0. The signed-manifest/allowlist and external helper-import tests
also passed with `-count=3 -tags e2e`. All lab commands cross-compiled for
linux/arm64; real node bundles contain amd64 and arm64 builds. Only amd64 ran.
Compose validation and shell syntax passed. Prefix teardown exited 0 with no
`fvpn-e2e-*` containers, volumes, networks or runtime image aliases remaining.
Shared neutral image cache tags were retained.

The persistent recovery config is created after enrollment and before agent
startup, root-owned mode 0600. The one-shot updater trigger is separate. This
fixes the missing boot config; SECOPS-09's positive counted-flow proof remains
blocked by the lease gate. AdGuard shares the node namespace and uses the exact
D-061 loopback field names. No product code, AT harness tree, module dependency,
projector-owned row or host networking was changed.

A discarded exploratory bring-up lost an unreferenced image after a shared
neutral tag moved. Build now adds a runtime prefix alias in the same build,
retaining the selected image while preserving the neutral shared build cache.
All three complete verification runs use these retainers and teardown removes them.
