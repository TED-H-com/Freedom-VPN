# Isolation run procedure

Use the assigned `w4-isolation` / `36000` binding. The mechanism container is
`fvpn-w4-isolation-lab`; this supersedes the legacy lab prefix in README.

```sh
make -f mk/at-isolation.mk at-isolation-lint
make -f mk/at-isolation.mk at-isolation
make go-lint
```

The run target builds the mechanism executable without privilege, then starts
the isolated privileged lab. It runs the mechanism controls, removes that lab,
creates a fresh assembled stack, runs the product cases and removes the stack.
Docker operations use only the assigned prefix. Host networking is untouched;
temporary build files use `.agent/tmp` on disk. A blocked or failed product AT
makes the target exit 2 after writing all four records. Mechanism passes never
promote a blocked product result.

`stack.py` calls the shared devstack lifecycle. A temporary Docker argument
wrapper substitutes task-owned cache tags for its neutral `fvpn-devstack-*`
tags. `down` removes those `fvpn-w4-isolation-cache-*` tags after the shared
script returns. `prefix.sh` has no `cache` token, so the shared script leaves
them, and the makefile leftover check would otherwise exit 1. All service
images retain their shared build definitions; the records contain the
immutable IDs of the running images. Python uses only its standard library.
The existing Ubuntu workflow already has the shared stack's Python dependency
and remains unchanged.

`driver.py` uses a Go overlay to add `driver.go.txt` to the shared `tests/e2e`
test package. This executes the original private `countedFlow`, `selectExit`
and `tracedFetch` helpers. It neither edits shared source nor copies the route
selection implementation. Vet accepts the overlay; pinned staticcheck analyzes
an ephemeral source copy under the owned directory because its parser cannot
open a virtual overlay filename. The copy is deleted on exit.

Setup uses `PrepareServingBaseline`, API-created DNS rules, groups, plans and
finite grants, native admission, and publication read-back. SQL is read-only.
No projector-owned row or limiter configuration is written. The only fleet
setting write is the AT13 admin API pair below.
The controlled DNS rule is nonempty, as in the members harness. No synthetic
probe is used as a member positive control.

Each negative embeds the same device's counted, CA-verified origin request.
Protected destination cases also verify a listening target from the node and
capture headers at node ingress and target ingress. The interface-change case
keeps at least ten protected-target connection attempts per second across a
dummy-interface change and a generation produced by an API capacity update.
All interface and route changes are confined to disposable containers.

Publication excludes B while retaining A, then disables and restores the exit.
Revocation observes established TCP/UDP fixtures, conntrack deletion, the next
serving generation and an admitted sibling. After that generation is serving,
`Stack.ObserveMemberIngress` injects one UDP/9 datagram from the revoked
overlay address and one from the still-admitted sibling onto `fvmb0`
(`tun-mirred`, link type 65534, source port 40000). The revoked source must
move `ctunku` by exactly one packet, leave every `u_`/`d_` counter still, and
stay off `fvup0` and the origin. The sibling must move its own `u_<hex>` by
exactly one packet and appear on both captures. The overlay calls that method
on the prepared `Stack`; it does not exec `fvpn-netlab` itself. The same hook
measures a forged `100.64.7.7` source for the AT12 unknown-source check.
Restart waits for a new serving heartbeat before checking recovery.

The finite-member expiry case checks the actual signed member deadline, then
stops both control and worker before it. This preserves the advertised route
while testing local expiry, with five post-deadline ingress captures and frozen
admission bytes. Both services are restarted afterward.

Whole-node expiry then `PUT`s `/api/v1/admin/fleet-config` to
`max_offline_lease_seconds=60` with reason
`TEST ONLY: AT13 whole-node expiry (isolation follow-up 4)`, using the strong
ETag, `If-Match` and `Idempotency-Key`. A capacity patch, the same desired-state
change the harness already makes, forces a renewal. The node must hold a 60s
lease with member grants. Control and worker are stopped before the renewal
lead. The node must keep forwarding until that signed deadline and return to
the baseline within one configured agent tick after it: ingress on `fvmb0`,
nothing on the uplink, admission counters flat. After control answers,
`86400` is restored with reason
`TEST ONLY: restore AT13 whole-node expiry ceiling (isolation follow-up 4)`.
Both `fleet_config.update` audit events are recorded. No host clock is changed.
No `fleet_config` row is written except through that API.

From publication until the first counted member byte, the runner samples
desired and applied generations, lease grant counts, observation block reasons
and the forward chain. The samples are `convergence-timeline.json`. Agent
follow-up 18 is on this `main`, so a grant-less same-generation sample must
show lease-check `error_code` `AWAITING_MEMBER_GRANTS` and block reason
`awaiting_member_grants`. The live
AT13 record repeats the SHA256 of `capture.go` concatenated with
`capture_review_test.go`; the singular/plural parser from follow-up 2 stays.

`AT*.json` are authoritative product records; `lab-AT*.json` and their sibling
captures/matrices are mechanism records. Each has a source hash, run ID, image
IDs and UTC timestamps. `summary.md` and `requests.md` describe the current run.
The #69 evidence lives in `history/follow-up-1/`. Diagnostic histories retain
their original dirty-source marker and are not the final acceptance run.

Follow-up 2's live run initially classified the finite-member expiry check as
blocked because its capture parser missed tcpdump's singular `1 packet captured`.
That original AT13 record is preserved in `history/follow-up-2-live/`. The corrected
parser accepts singular/plural summaries and rejects capture loss or an interface
disappearing. Its unit tests and the offline review of that archived capture
established five ingress packets, zero egress packets, five timeouts, a retained
exit route and frozen counters. That correction stays in force. Follow-up 3 did
not edit `capture.go` or `capture_review_test.go`. Its live AT13 record repeats
`SHA256(concatenated capture.go, capture_review_test.go)=d4cd4dfd0957cace3ef9cd63168a4653c75ac1d0ba07d5c5ca1af00603f38d53`.

Follow-up 3's assembled run is `49799a38-1980-4c0f-a527-725646230eee`, source
`4d84f0a651e60ae84f168a8a33248503a50cc94a`, UTC `2026-10-30T19:43:32Z` through
`2026-10-30T19:57:57Z`. AT10 passed, including a positive for every destination
case. Short whole-node expiry passed through the fleet-config API, and
finite-member expiry was re-measured in the same stack. That record is archived
under `history/follow-up-3/` and is not the current run.

Follow-up 4's assembled run is `fa0074cc-3b1a-403d-b8c2-3a3439fe2b7e`, source
`491a307dd505e33b14dc0e993aa7e86b28e82f68`, UTC `2026-10-31T00:27:07Z` through
`2026-10-31T00:31:43Z`, on main `205633f`. Serving publication fail-closed
`FLOWTABLE_BYPASS` on Docker's `ip nat` `POSTROUTING` `xt` SNAT for
`127.0.0.11`. No counted byte, no convergence sample, and no fleet-config PUT
were recorded. AT10–AT13 stay blocked. The aggregate exited 2. `summary.md`
is the measured record. When publication fails, the runner reads
`nft -j list ruleset` and stores the foreign chains on check `serving setup`.
It does not write nft.

`at-isolation-review-captures` rewrites the AT13 record in its review directory.
Follow-up 3's root AT13 is the live measurement; do not pass it through that
target. To reproduce the follow-up 2 offline review, point the test at a copy
of `history/follow-up-2-live`:

```sh
FVPN_AT_REVIEW_DIR="$PWD/docs/e2e/at/isolation/history/follow-up-2-live" \
  go test -count=1 -run '^TestAT13_ReviewCapturedExpiry$' ./tests/e2e/at/isolation
```
