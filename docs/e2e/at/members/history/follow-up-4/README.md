# Wave 4 member acceptance — Follow-up 4

Run only the assigned `w4-members / 24000` stack. [RUN.md](RUN.md) records both
fresh attempts; [summary.md](summary.md) lists current AT outcomes and
[requests.md](requests.md) names the remaining owners. Prior evidence is
preserved under [history/follow-up-3](history/follow-up-3/RUN.md).

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
make -f mk/at-members.mk at-members-readback
FVPN_AT_MEMBERS_OBSERVE_ONLY=1 make -f mk/at-members.mk at-members-readback
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
```

The full target is intentionally non-green when any complete AT is blocked.
The readback targets are partial observations on the same images and incarnation;
their successful exit cannot pass a full AT. Earlier assertions remain in
`prior_checks` or the archived complete attempt, with their original UTC times.

D-050 applies throughout: signed releases use `PrepareServingBaseline`; groups,
plans, grants, publication and devices use product APIs and production admission.
Native login uses the shared Dex walker. Two barrier-synchronized goroutines
exercise the real device adapter for the last slot. SQL reads projections; only
synthetic audit retention inputs are inserted directly.

Billing uses real checkout sessions, authenticated provider snapshots and exact
signed hook bytes for every event, including created and failed. Replays are
paced two seconds apart, asserted once and never retried after a 429. No limiter
setting, invoice, subscription or entitlement projection is changed by the test.

Forwarding records contain native route selection, route interface, node named
counter snapshots and concurrent bounded packet observations inside the member,
node and origin containers. The observer keeps only counts by interface and
direction for the controlled TCP connection; no packet body, URL history or
credential is retained. Both a CA-verified origin response and increases in the
specific device's upload/download counters, plus authenticated origin receipt
from the selected node's SNAT address, are required for counted forwarding.
See [forwarding-observation.md](forwarding-observation.md).

DNS policy revisions reuse one active definition. Node-mTLS document digests
and typed `dns_policy_evidence` distinguish publication from actual application.
The scheduled feed experiment tests `attribution:null`, rejects blank notes,
then observes a real failed refresh and retained last-good content. Loopback DNS
is explicitly separate from native member DNS; the stock client disables DNS.

Records include run UUID, source/build/image digests, UTC timestamps, commands,
expected/actual results, policy revisions and incarnation. `complete:false`
means an incremental record, never a completed AT. Raw logs, provider bodies,
signatures and credentials stay out of public evidence.

[OPERATORS.md](OPERATORS.md) retains exact **needs operator run** scripts for
Windows/macOS/iOS/Android, physical arm64 and second physical NAT.
[PROCEDURES.md](PROCEDURES.md) describes the complete acceptance matrix.

```sh
make -f mk/at-members.mk at-members-lint at-members-test at-members-build
make go-lint
go vet -tags e2e ./tests/...
```

The unchanged workflow uses the existing shared Portal browser dependency,
checks the owned packages and tears down only this prefix.
