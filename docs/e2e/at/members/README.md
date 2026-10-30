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
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
```

Use a fresh stack. The suite authenticates through its IdP, writes finite grants
through HTTP, enrolls the installer node through the documented token-file hook,
uses a setup-key peer only as a negative admission control, submits real signed
webhooks, activates DNS policy, pauses/restores its resolver containers and runs
the guarded retention purge. It never inserts member identity proofs or device
bindings. TEST ONLY plan/group catalogue rows and synthetic old/young audit rows
are the only direct fixture writes. It does not run the whole shared scenario
runner, which also revokes nodes and performs update faults.

`at-members` exits nonzero for a blocked or failed AT. This is intentional: a
successful HTTP command or an unavailable positive control cannot pass a full
acceptance case. `tests/e2e.Write` renders the evidence; JSON adds step UTC times,
commands, expected/actual values, source/build/image digests, applied generation
and node incarnation. Only aggregate, synthetic or allowlisted fields are
published. Raw provider bodies, database dumps and container logs stay in memory.

| AT | Status | Evidence | Remaining operator requirement |
|---|---|---|---|
| AT01 | blocked | [JSON](AT01.json), [page](AT01.md) | four native platforms |
| AT02 | blocked | [JSON](AT02.json), [page](AT02.md), [trace](AT02-TRACE.md) | none substituted for missing lab admission |
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
