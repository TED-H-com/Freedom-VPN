# Wave 4 member acceptance

**Blocked at startup; no assigned AT has passed.** The prescribed fresh-stack
command rejected port base `19000` before creating any containers. This is a
partial hand-off, not a completed live acceptance suite. It needs devstack
follow-up 2: reconcile the assigned ports with the merged slot validator.

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=19000 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=19000 make -f mk/e2e.mk e2e-down
```

Actual startup diagnostic:

```text
fvpn-e2e-prepare: port base must be 18000 plus a multiple of 6000
make: *** [mk/e2e.mk:39: e2e-up] Error 2
```

`at-members` reproduces the owner's prepare-command rejection using private disk
scratch, writes a JSON record and Markdown page for each AT, and exits nonzero.
It uses the shared `tests/e2e.Write` evidence format and adds typed provenance.
It does not start a second lab, copy the runner, change ports, or report the
startup rejection as a passed acceptance test. A future successful prepare also
remains blocked until the live cases are implemented and executed. The recorded
prepare executable hash is **not** a digest of a running service image. Missing
image digests, policy revision and incarnation are explicitly unavailable.

| AT | Lab status | Additional operator work | Evidence |
|---|---|---|---|
| AT01 | blocked | needs operator run: Windows, macOS, iOS, Android | [JSON](AT01.json), [page](AT01.md) |
| AT02 | blocked | real gateway denial still required | [JSON](AT02.json), [page](AT02.md), [binding trace](AT02-TRACE.md) |
| AT03 | blocked | none substituted for missing lab hooks | [JSON](AT03.json), [page](AT03.md) |
| AT04 | blocked | all-node flow-stop timing still required | [JSON](AT04.json), [page](AT04.md) |
| AT05 | blocked | needs operator run: physical arm64, all three sources | [JSON](AT05.json), [page](AT05.md) |
| AT08 | blocked | pending/applied UI and forwarding still required | [JSON](AT08.json), [page](AT08.md) |
| AT09 | blocked | needs operator run: platforms, second physical NAT | [JSON](AT09.json), [page](AT09.md) |
| AT20 | blocked | actual resolver and rebinding traffic still required | [JSON](AT20.json), [page](AT20.md) |
| AT21 | blocked | needs operator run: Private DNS and browser DoH | [JSON](AT21.json), [page](AT21.md) |
| AT22 | blocked | candidate config, backup and retention still required | [JSON](AT22.json), [page](AT22.md) |

[PROCEDURES.md](PROCEDURES.md) specifies the tests to implement after the shared
hooks land. [OPERATORS.md](OPERATORS.md) gives the platform run script and member
guidance. [BLOCKERS.md](BLOCKERS.md) gives owner, function, reproduction and
expected behaviour for each missing prerequisite. Historical evidence in
`docs/e2e/runs/2026-10-29` belongs to the earlier runner; it is not this run.

Offline verification:

```sh
make -f mk/at-members.mk at-members-lint at-members-test
make go-lint
# Set TMPDIR and GOTMPDIR to private disk scratch under .agent/tmp first.
go vet -tags e2e ./tests/...
```

The workflow runs lint on relevant PRs. Manual dispatch attempts the assigned
stack, records the non-green startup gate, tears down that prefix and uploads
only public evidence. It does not upload `.agent` or generated secrets.
