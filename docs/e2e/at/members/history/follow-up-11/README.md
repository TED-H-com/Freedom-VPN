# Member acceptance

Assigned instance: `FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000`.
Current outcomes: [summary.md](summary.md), [native transport and DNS rows](NATIVE.md), [run ledger](RUN.md), and the ten current AT JSON/Markdown pairs. The full target exits nonzero while a complete case is blocked. [D077.md](D077.md) indexes the production device re-authorization calls, bodies, operation IDs and fence counts.

```sh
make -f mk/at-members.mk at-members-lint
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 FVPN_E2E_TWO_RESOLVERS=true E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
make -f mk/at-members.mk at-members-test
python3 tests/e2e/at/members/disktmp.py make go-lint
mkdir -p .agent/tmp
TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp" CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...
make -f mk/at-members.mk at-members-build
```

Use a fresh stack. The readback target supplements the same run and verifies image/incarnation identity; it does not overwrite primary failures or pass blocked cases. The primary deadline is 50 minutes (Go test limit 55 minutes); the manually dispatched workflow reserves 90 minutes for startup, scheduled readback and cleanup. Disk scratch is private under the worktree's `.agent/tmp`; no host tmpfs is used.

Signed releases, native peers and second exits reuse the shared lab hooks. Groups, plans and grants use the product APIs; quota contention uses the real production adapter. D-077 adds only the explicitly authorized restore-fence SQL input. D-081 uses the exported `ReplaceFiniteGrant` (#284); exact safe HTTP bodies and worker IDs accompany the projection/audit evidence. No identity, device, entitlement, binding or observation projection is repaired.

Forwarding requires a controlled origin receipt and the named device counters; [forwarding-observation.md](forwarding-observation.md) explains the boundary. Operation acceptance remains distinct from observed completion, including the existing operation returned by fenced re-grants. Billing replays are paced and asserted once. The new DNS rows use the native OS stub and exactly two product-issued resolvers; their TEST ONLY distribution is explicit. Credentials and raw provider signatures are not published.

[BLOCKERS.md](BLOCKERS.md) and [requests.md](requests.md) retain F4-03/04/05/06/07/10. [OPERATORS.md](OPERATORS.md) retains exact **needs operator run** procedures for Windows/macOS/iOS/Android, physical arm64 and second NAT. The complete matrix is in [PROCEDURES.md](PROCEDURES.md).

[Follow-up 7 final run](history/follow-up-7/RUN.md) is byte-identical under D-098, with [checksums](history/follow-up-7/SHA256SUMS). [Follow-up 6](history/follow-up-6/RUN.md) and the [interrupted fixture](history/follow-up-7/attempt-1/RUN.md) and [meter-collision](history/follow-up-7/attempt-2/RUN.md) attempts remain archived. Current pass claims come only from the final run.

Follow-up 11 enables the detached TEST ONLY enforcement observer **before
enrollment**. Every AT02 persistent sample retains the independent counter and
transport-status `read_diagnostic`, including failed reads, plus same-boot
CLOCK_MONOTONIC brackets and both completed producer records. The complete
[producer stream](AT02-enforcement-timing.json) is checked for sequence gaps,
missing records and zero/unordered clocks. No failed read is retried or removed;
`finishPersistent` retains its original every-read-success condition.

Native DNS results retain the shared `dns_trace` unchanged. Unique metadata
matches identify a responder; multiple matches stay ambiguous. Observed
authority exchanges prove misses, while absence supports only an inferred
positive hit. Negative policy cache state remains unknown unless independently
attributable. Authority serial sampled after a query is not a cached A record's
serial, and native probe milliseconds are distinct from the wrapper bounds.

D-129 closes F10-04 at the unchanged sequence's product-issued generation;
full AT09 still needs all selector rows. F10-02/F10-03 remain historically
failed and unattributed. [Follow-up 10](history/follow-up-10/RUN.md) is
byte-identical with [35-file checksums](history/follow-up-10/SHA256SUMS).
D-132 permits product pins only into `history/`; the root is living evidence.
The [interrupted admission attempt](history/follow-up-11/attempt-1/RUN.md)
stopped in AT05 setup before AT02/AT20 and is retained separately.
