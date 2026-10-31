# Member acceptance

Assigned instance: `FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000`.
Current outcomes: [summary.md](summary.md), [run ledger](RUN.md), and the ten current AT JSON/Markdown pairs. The full target exits nonzero while a complete case is blocked. [D077.md](D077.md) indexes the production device re-authorization calls, bodies, operation IDs and fence counts.

```sh
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 FVPN_E2E_TWO_RESOLVERS=true E2E_RESET=1 make -f mk/e2e.mk e2e-up
make -f mk/at-members.mk at-members-lint
make -f mk/at-members.mk at-members
FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 make -f mk/e2e.mk e2e-down
make -f mk/at-members.mk at-members-test
make go-lint
mkdir -p .agent/tmp
TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp" CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./...
make -f mk/at-members.mk at-members-build
```

Use a fresh stack. The readback target supplements the same run and verifies image/incarnation identity; it does not overwrite primary failures or pass blocked cases. The primary deadline is 50 minutes (Go test limit 55 minutes); the manually dispatched workflow reserves 90 minutes for startup, scheduled readback and cleanup. Disk scratch is private under the worktree's `.agent/tmp`; no host tmpfs is used.

Signed releases, native peers and second exits reuse the shared lab hooks. Groups, plans and grants use the product APIs; quota contention uses the real production adapter. D-077 adds only the explicitly authorized restore-fence SQL input. D-081 uses the exported `ReplaceFiniteGrant` (#284); exact safe HTTP bodies and worker IDs accompany the projection/audit evidence. No identity, device, entitlement, binding or observation projection is repaired.

Forwarding requires a controlled origin receipt and the named device counters; [forwarding-observation.md](forwarding-observation.md) explains the boundary. Operation acceptance remains distinct from observed completion, including the existing operation returned by fenced re-grants. Billing replays are paced and asserted once. The new DNS rows use the native OS stub and exactly two product-issued resolvers; their TEST ONLY distribution is explicit. Credentials and raw provider signatures are not published.

[BLOCKERS.md](BLOCKERS.md) and [requests.md](requests.md) retain F4-03/04/05/06/07/10. [OPERATORS.md](OPERATORS.md) retains exact **needs operator run** procedures for Windows/macOS/iOS/Android, physical arm64 and second NAT. The complete matrix is in [PROCEDURES.md](PROCEDURES.md).

[Follow-up 7 final run](history/follow-up-7/RUN.md) is byte-identical under D-098, with [checksums](history/follow-up-7/SHA256SUMS). [Follow-up 6](history/follow-up-6/RUN.md) and the [interrupted fixture](history/follow-up-7/attempt-1/RUN.md) and [meter-collision](history/follow-up-7/attempt-2/RUN.md) attempts remain archived. Current pass claims come only from the final run.
