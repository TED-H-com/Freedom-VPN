# Follow-up 21 — installer input, DNS lifetimes and fixture attribution

TEST ONLY. D-098 was first: commit `970a1305` pins all **114** follow-up 20 files
in [followup20-archive.sha256](../../../followup20-archive.sha256), with a
byte-identical private disk copy. Follow-ups 16–20, including `cce1a688`, are
unchanged. No product code or acceptance-lane artifact is modified.

## Shared installer wrapper

[Fresh run 33b9621a](33b9621a-41d7-49fb-bc27-1c3f05e4ab44/installer-loopback-wrapper.md)
passes **3 / 0 / 0**, actual process exit **0**, source `5bc41666`.
`RerunInstaller` now supplies the documented explicit TEST ONLY `--allow-loopback`
through its spec, only for its exact existing local proxy. Install returns **0**,
`installed`. The generated reporting URL is
`https://127.0.0.1:9444/api/v1/agent/update-results`, with exact origin policy,
loopback opt-in and the generated server CA/client identity paths.

Generated config SHA-256 before and after the probes:
`14a8975a8a36df9558b9b957e07613476e43de351281eadaae4657e860f06164`.
No-client-certificate POST returns **403**; a deliberately invalid empty receipt
with the generated certificate/key and CA reaches central validation (**400**).
No successful receipt is manufactured by this probe. Updates 11 retains the
independent real automatic-delivery proof and confirmed regular release-root
closure (installer 27/27, regular AT23–AT27 5/5).

The [initial attempt 52d682cd](52d682cd-83b6-419c-9fa0-9da6d2ecc027/admin-login.md)
is **aborted**, process exit **1**, after one login record. `PrepareInstallerNode`
cannot attach to the origin network after the DNS campaign retained node3:
both existing fixtures allocate origin slot `.24`. The failure is preserved.
The successful measurement uses a fresh stack, with no allocation or production
change. Each documented campaign starts and tears down independently.

## Whole-sequence DNS request history

[Final run cba88bbc](cba88bbc-a7bf-49f8-9dab-3182123d02aa/dns-negative.md), source `5bc41666`,
retains the unchanged DNS assertions and the separate
[history-integrity row](cba88bbc-a7bf-49f8-9dab-3182123d02aa/dns-history-integrity.md) (`pass`).
It records **36 / 36** complete query traces,
**85** distinct authority requests and
**70** identities retained across multiple query snapshots.
Each request has a start before dispatch, a stable random identity and its
later completion/error. Earlier unresolved requests survive into later
snapshots, including across explicit relay process-stream boundaries.

The [export contract](../../../scenarios/harness-exports.md#follow-up-21--installer-input-and-dns-lifetime-correlation-test-only)
retains cumulative authority history, outstanding IDs at query start/end/read,
per-role read times and all stream IDs. Complete receipts do not mean there was
no older outstanding fetch, restart or cache invalidation. The backend's
[fetch-versus-fallback and F9-INFLIGHT rule](../../../../dns/BACKEND.md#215-f10-03--positive-cache-and-ttl-measured-2026-10-31)
is adopted. Hit wording remains inferred. A separately sampled authority SOA
is never attached retrospectively to a positive cached answer.

Both real resolvers retain before/after live read-back: **4096-byte (4 KiB)**
cache, zero minimum/maximum TTL overrides, optimistic caching **false**, API
upstream mode **empty** (load balancing), upstream timeout **10 seconds**,
zero fallback upstreams, NXDOMAIN / blocked TTL **5** (TEST ONLY). The historical
4 KiB fixture override is deliberately retained for comparability; the pin's
4 MiB default was not silently substituted. The relay's existing 2-second
exchange bound is separate from the resolver timeout. No cache clear, TTL,
timeout, production default or query assertion is changed.

The [first DNS run 99027e5d](99027e5d-ae5e-40d4-b595-7b550f89a52d/dns-negative.md)
passes its **12 / 0 / 0** functional rows, but its new trace reader marks the
intentional fallback restarts incomplete. That artifact is unchanged. The final
reader checks continuity within each explicitly retained process stream and
keeps unresolved requests; it does not hide restarts. Five focused unit checks
cover scope, privacy, late replies, missing/partial/duplicate receipts and
pending requests across restart. The final live diagnostic row enforces the
completeness claim separately from DNS behavior.

The focused runs do not reproduce the backend's deliberately delayed
3.5-second F9-INFLIGHT experiment or prove private cache insertion. They do not
reconstruct historical F10-03. Neither a TTL fix nor new whole-AT acceptance is
claimed. The fallback observation intentionally has incomplete receipts while
the primary relay is stopped; it is separate from the complete matrix.

## Members and updater handoffs

[Members fixture analysis](../../../scenarios/followup21-members.md) and the
[pinned original inputs/timings](../../../followup21-members.json) locate F11-01
only between member inner egress and node observable inner ingress: encrypted
transport sub-hops were not captured. F11-03 retains all twelve generation-13
Applies: eight `install_ok=false` calls exceed the unchanged 100 ms guarded
installation bound (184–304 ms), do not release the guard, and reach no
verification records. Raw command errors, effective resource pressure, full
historical model and injection monotonic brackets are absent. No fixture or
health criterion was changed to pass either members row.

[Stopped-producer scenario](../../../scenarios/updater-negative.md) classifies
updater F18-01's **1 / 1 / 2** as future negative D-125 coverage. The old failed
candidate-success row is unchanged and is not a passing upgrade. All D-102
answers, including members' populated-grant readiness prerequisite and updates'
remaining outside-owned offline-install documentation request, are in
[RESPONSES](../../../RESPONSES.md).

## Reproduction and gates

Run each campaign on its **own fresh stack**, with disk TMPDIR/GOTMPDIR under
`.agent/tmp`, `CGO_ENABLED=0`, `GOTOOLCHAIN=local` and `GOCACHE` from `go env GOCACHE`.
Before up/live, check `pgrep -af 'gates.sh|mk/at-'` and wait while two other
integration suites are active; check again after startup.

```sh
FVPN_E2E_TWO_RESOLVERS=true make -f mk/e2e.mk e2e-up E2E_RESET=1
FVPN_E2E_FOLLOWUP21=1 go test -tags e2e -count=1 -v -timeout 20m ./tests/e2e -run '^TestE2E_Followup21DNS$'
make -f mk/e2e.mk e2e-down
```

For the installer, repeat fresh up/down and select
`^TestE2E_Followup21Installer$` with the same foreground/tag/count flags.
The [scenario ledger](scenarios.md) preserves each original status;
[gates.md](gates.md) records actual exits, including the aborted setup.
All required final offline gates pass in order, including committed-evidence
hygiene, handover, whole-module lint and amd64/arm64 builds. The corrected
final audit verifies zero owned Docker containers/networks/volumes/images,
all 114 archive pins, immutable request joins and local documentation links.
The first supplemental scope audit used a direct comparison to advancing main;
its false scope failure and corrected `main...HEAD` audit are both retained. Live execution is amd64;
arm64 cross-compilation is not physical arm64 acceptance. The shared PostgreSQL
is not reconfigured, and final owned Docker inventory must be zero.
