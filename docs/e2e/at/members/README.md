# Wave 4 member acceptance

The suite runs real HTTP, database, NetBird, DNS and container probes on the
assigned devstack. **No complete assigned AT has passed.** Each JSON/page records
successful subchecks separately from observed missing prerequisites. Native
platforms and physical hardware remain **needs operator run**.

The orchestrator corrected the slot assignment to `24000`; the startup gate
accepts it. That earlier assignment error is resolved and is not a lab defect.

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
# Run on the same stack even when full ATs are blocked:
make -f mk/at-members.mk at-members-readback
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
```

Use a fresh stack. The suite uses the shared Dex login walker and the lab's
native `user-auth` entrypoint. Additional quota contenders reuse the unchanged
member image, entrypoint and capabilities. Finite grants and revocations go
through HTTP; two real-adapter goroutines race for the final slot. Every
identity/device/entitlement/subscription output is a product projection (D-050).
Direct fixtures are limited to TEST ONLY plan/group inputs and synthetic audit
rows. No projections or ready/serving states are hand-written.

Follow-up 1 proves the native public-subject mapping, quota transaction,
cross-owner existing-device rejection, slot reuse, and real-binding usage
receipt/replay. Checkout, webhook replay/order, real grant expiry and suspension
checks also pass. Applied forwarding, paid subscription truth and node DNS
delivery remain blocked. `at-members-readback` records exact node-mTLS readiness
gates, then tests the D-054 feed worker using self-authored controlled-origin
bytes and a real 300-second TEST ONLY cadence. It restores generated DNS settings
and restarts only this stack's worker afterwards. It supplements the existing
run only when image digests and node incarnation match.

The initial null-attribution experiment exposed B9; its explicit-attribution
positive control and existing last-good state were inspected with
`FVPN_AT_MEMBERS_FEED_READBACK=1 make -f mk/at-members.mk at-members-readback`.
The normal feed fixture now supplies attribution. The optional inspection is a
diagnostic, not a full-AT pass. See [RUN.md](RUN.md) for exact outcomes.

`at-members` exits nonzero for a blocked or failed AT. This is intentional: a
successful HTTP command or an unavailable positive control cannot pass a full
acceptance case. `tests/e2e.Write` renders the evidence; JSON adds step UTC times,
commands, expected/actual values, source/build/image digests, applied generation
and node incarnation. Only aggregate, synthetic or allowlisted fields are
published. Raw provider bodies, database dumps and container logs stay in memory.

| AT | Status | Evidence | Remaining operator requirement |
|---|---|---|---|
| AT01 | blocked | [JSON](AT01.json), [page](AT01.md) | four native platforms |
| AT02 | blocked | [JSON](AT02.json), [page](AT02.md), [trace](AT02-TRACE.md) | gateway positive control still required |
| AT03 | blocked | [JSON](AT03.json), [page](AT03.md) | none substituted for missing provider snapshots |
| AT04 | blocked | [JSON](AT04.json), [page](AT04.md) | rendered screen and every-node flow stop |
| AT05 | blocked | [JSON](AT05.json), [page](AT05.md) | physical arm64, three sources |
| AT08 | blocked | [JSON](AT08.json), [page](AT08.md) | applied publication and group traffic |
| AT09 | blocked | [JSON](AT09.json), [page](AT09.md) | platforms, second physical NAT |
| AT20 | blocked | [JSON](AT20.json), [page](AT20.md) | none substituted for missing DNS fixtures |
| AT21 | blocked | [JSON](AT21.json), [page](AT21.md) | Private DNS/browser DoH |
| AT22 | blocked | [JSON](AT22.json), [page](AT22.md) | candidate config and retention inventory |

[PROCEDURES.md](PROCEDURES.md) gives the complete remaining acceptance script;
[OPERATORS.md](OPERATORS.md) gives exact platform actions and draft member
guidance. [BLOCKERS.md](BLOCKERS.md) identifies owner paths and requested fixes.
Older `docs/e2e/runs` evidence is not presented as a result of this run.

```sh
make -f mk/at-members.mk at-members-lint at-members-test
make go-lint
go vet -tags e2e ./tests/...
```

Targets use private disk scratch. The workflow lints PRs, cross-compiles the
probe for both architectures, and runs the assigned live stack on manual
dispatch. It removes only its prefix and uploads only public AT evidence.
