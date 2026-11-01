# Follow-up 14 execution note

One assembled ten-case run uses compiled harness `e9a215f3`, product baseline
`8f3055b9` (#369 and #370). See [RUN.md](RUN.md) for the final rows and gates.
The subsequent harness correction moves the new routed hop after the existing
eth0-only relay fault; it is not live-qualified by this run. No second run is
claimed or authorized here.

`assembled.py` performs admission checks before and after startup and invokes
`make -f mk/at-members.mk at-members` in the foreground, then tears down through
Python `finally`. It clears the retired follow-up 12 skip flag. All scratch is
on disk through `disktmp.py`; the actual cache is read with `go env GOCACHE`.

The original AT20 query is captured once, after two-resolver configuration and
before readiness/listener/RR/cache operations. Both complete resolver lifecycle
histories, wire IDs, reply digests, capture receipts and payload-free headers are
retained. Positive IDNA UDP/TCP controls follow the RR matrix. Raw captures and
exact counter parser errors remain private. These commands derive evidence from
that run's retained files and send no traffic:

```sh
python3 -B tests/e2e/at/members/followup14_evidence.py
python3 -B tests/e2e/at/members/followup14_receipts.py
python3 -B tests/e2e/at/members/followup14_transitions.py
```

Command-lifecycle absence is unavailable. Present records must have valid
Apply/command bounds and join the original producer identity. Withdrawal reads
retain non-nil error results and their `parse_failure` receipts; no retry replaces
a failed sample. Repeated chain declarations are an owner diagnosis, not permission
to change parser or acceptance predicates.

# Member acceptance execution procedures

The JSON records are the authority for executed subchecks. These procedures
specify full acceptance, including steps blocked by the observed prerequisites
in [requests.md](requests.md). They do not authorize editing the shared lab or
substituting mock identity, payment or forwarding evidence.

## Common setup and evidence (§19.1)

Use only the assigned `w4-members` stack, base `24000`, and
`.agent/e2e-state-w4-members`. Bring it up fresh once. Do not use `e2e-test` after
fixture creation: that target tears down and recreates the stack. Do not run the
existing whole scenario runner as a setup hook: it also performs updates,
control-stop and node revocation. Use `e2e.PrepareServingBaseline` before any enrollment token, then the D-057 group/policy/publication APIs. Keep one active DNS definition and revise it through the policy API; use a controlled positive allow rule in the initial profile.

Provision TEST ONLY members A (active, group A), B (active, group B), C
(pending/expired). Prices, grace, full-refund rule, rate policy and blocked DNS
TTL must be explicit TEST ONLY configuration. Use device limit 3. Keep every
scenario's events, devices and subscriptions distinct. No financial rule is
inferred from a payment amount. Use wall-clock expiry; never change host time.

Capture source commit and dirty-tree state; actual image IDs/RepoDigests of
control, worker, node, member, NetBird, resolvers, IdP, payment and endpoints;
node incarnation and architecture; applied policy generation/revision; UTC start
and finish. `docker inspect --format '{{.Image}}' fvpn-w4-members-node` reads the
node image ID. Inspect each explicitly named container only; do not archive
full Docker inspect output because it includes environment/configuration.

For each step retain request method/route, status/error code, pseudonymous
operation and resource IDs, expected/actual comparison, monotonic elapsed time
and UTC anchors. Keep decimal sequences/revisions as strings. Never publish
Authorization/cookies/CSRF values, setup/enrollment codes, private keys, DSNs or
full webhook bodies. Capture only approved evidence fields. A 202 proves intent;
wait for required operation steps, external readback and actual traffic before
claiming application. Failed positive controls block downstream denial claims.

Every side-effecting admin/member command uses a fresh UUID Idempotency-Key;
intentional retries reuse the original key. Fetch ETag and supply If-Match on
mutable endpoints required by the contract. Authentication is through the real
IdP and `/auth/login?role=member|admin`, never invented `/api/v1/auth` routes.

## AT01 — identity and first use (§7.3, §9.1)

1. Log member A into Portal and the NetBird native user client through the same
   test issuer, two distinct client audiences. Retain issuer/scope pseudonyms,
   stable-claim proof method and member/peer/device join evidence. A pair of
   equal subject strings or equal emails is insufficient proof.
2. Assert one member and one authorized device/binding after discovery; compare
   server list with NetBird user/peer readback. Before admission, a positive
   network control must work while member forwarding is denied.
3. Publish a ready exit for A; record pending operation then matching ACK,
   route and data-path evidence. Check `GET /api/v1/member/exits`, native route
   list and explicit selection. Reach only the controlled echo endpoint and
   assert the recorded egress belongs to the selected exit.
4. Sign out of Portal using its CSRF-protected logout. Confirm `/auth/session`
   unauthenticated and private endpoints denied. Record native tunnel behaviour
   separately; browser logout is not specified as device revocation. Sign out
   of the native client, then reconnect using SSO and verify ownership/quota.
5. Move the member between the two owner-provided networks, registering Go `t.Cleanup`
   to restore its original network. Recheck selected exit, member/device
   identity, route, DNS and echo. Never change host routes.
6. `docker pause fvpn-w4-members-member`, wait 10 seconds, then
   `docker unpause fvpn-w4-members-member`; install cleanup before pausing.
   Record recovery time and the same checks. Container pause is only a Linux
   simulation; run [OPERATORS.md](OPERATORS.md) on all four real platforms.

## AT02 — final slot, ownership and revoke (§9.4)

1. Admit two of A's three allowed devices through the verified peer path. Start
   two more native user peers without access. B has a distinct admitted device.
   Check the binding trace before attempting the race.
2. Through the requested test-only admission seam, create two goroutines, each
   with its own UUID command key, and have both wait on one closed start channel.
   Each calls `ObserveAndAuthorize` for its own actual peer. Collect both results
   before asserting: exactly one succeeds and one returns `ErrLimit` (mapped
   `DEVICE_LIMIT_REACHED`, 409), authorized count exactly 3, one admission
   operation/outbox/binding for the winner, none authorizing the loser.
3. Use A's real session to list devices. B's device fields must be absent. There
   is no GET-by-ID member-device endpoint: do not claim a route-level 404 proves
   ownership. List filtering plus `DELETE /api/v1/member/devices/{B-device}`
   yielding 403/404, no B fields in the error and no B state change prove it.
4. Establish traffic from the winner through the same exit and endpoint, then
   attempt it from the loser. Count successful response bytes and gateway
   admitted/denied outcomes; a timeout from both clients is an invalid control.
5. Revoke one old A device through the real DELETE route; its existing and new
   flows must stop and its NetBird authorization must withdraw. Authorized
   count becomes 2. Admit the loser once: count becomes 3. An additional peer
   is rejected. Repeating DELETE with its original key releases no extra slot.
   Retain the revoked tombstone and verify it cannot silently reauthorize.

## AT03 — payment replay and ordering (§9.3)

Use the provider fixture's authoritative state hook, not direct entitlement
updates. Freeze one paid invoice interval `[start,end)` in that provider state.
The webhook signature covers the exact original bytes.

1. Submit the identical signed paid event ten times to
   `POST /api/v1/webhooks/stripe`. Expect durable acknowledgement and one inbox
   row for `(provider,event_id)`. Wait for worker processing, then compare
   subscription reconciliation snapshot, entitlement history and policy outbox.
   Paid/effective end must equal the original interval, not ten added periods.
2. With a separate subscription fixture per row, deliver all orders:

   | Run | Delivery order |
   |---|---|
   | 1 | created, paid, failed |
   | 2 | created, failed, paid |
   | 3 | paid, created, failed |
   | 4 | paid, failed, created |
   | 5 | failed, created, paid |
   | 6 | failed, paid, created |

   Keep the authoritative current invoice paid. Wait for processing after each
   delivery and assert the same final paid interval in every run. Replay a late
   failure after a newer paid renewal and verify no regression. Record semantic
   entitlement revisions separately from inbox/reconciliation timestamps.
3. For C without paid rights, load the allowlisted return page with forged
   `success=true`/session query fields and refresh it. `GET /member/me` must
   still deny service; no billing inbox, paid interval or gateway access changes.
4. Send the same event with one changed signature byte. Expect contract error,
   no new durable inbox or entitlement mutation. Keep the raw secret/signature
   out of evidence. A valid control event must still work.
5. Prove gateway authorization follows the one valid entitlement and remains
   denied for C. UI changes and HTTP 202 alone do not prove correct opening.

## AT04 — cancellation, refunds, expiry and suspension (§9.2, §16.5)

For each row use a fresh subscription/member fixture and two serving exits.
Record TCP and UDP receive counters once per second for at least 10 seconds
before changing rights. Use sustained flows plus fresh connection attempts.

| Transition | Expected service behaviour |
|---|---|
| Cancel at period end | Preserve paid rights until original paid deadline; stop on expiry. |
| Full refund | Apply the explicitly configured refund rule to the original invoice period; never subtract an arbitrary month. |
| Partial refund | Preserve period unless an explicit product rule says otherwise; retain reconciliation evidence. |
| Natural finite expiry | Stop at the effective service/policy deadline; no new login needed. |
| Manual suspension, then valid paid renewal | Remain suspended; payment cannot clear the administrator flag. |
| Finite admin grant | Only the authorized interval contributes; suspension still takes priority. |

At event commit capture UTC `t0`, source and entitlement revision. For each
node capture receipt of desired state, applied revision, last received flow
byte, first failed new connection and ten subsequent seconds without admitted
traffic. Both old/new TCP and UDP must stop within the configured target (TEST
ONLY 60 seconds, the §16.5 starting target). Record node count and every node's
result; disconnected nodes are unconfirmed and tested against their own lease,
not counted as online ACKs. Compare Portal entitlement/actions against
`GET /api/v1/member/me`. A suspended label alone is insufficient.

## AT05 — three join sources (§6.1–§6.4)

Use owner-provided preinstalled-image, cloud-init and operator-installer
containers on amd64. Each starts with no generated identity and a distinct
single-use token. Default token validity is 7 days for preinstalled and 15
minutes for installer; record actual issue/expiry without the token value.

Capture package and agent hash, protocol version, OS/architecture, enrollment
request and incarnation pseudonyms for each. Verify same request replay returns
the same incarnation and no second peer; each bootstrap is delivered once.
After automatic report, require registered/ready/**unpublished**, no member
forwarding, and no additional human admission action. Through the same Admin
API/UI classify name, region, member groups, then publish and wait for all
required readbacks/ACK/data-path evidence. Repeat on physical arm64 per
OPERATORS.md; cross-compilation is not runtime evidence. If emulation is later
used, explicitly record emulator/version, host architecture and its limits.

## AT08 — publication, groups, disable (§5.2, §7.2)

Begin with an unpublished ready node. Create groups through `POST /admin/groups`, set plan `exit_group_ids`, and
publish only for A using `PublicationRequest.group_ids` in the contract
publication command and current ETag. Record the 202 operation while pending;
poll until required policy/route/data-path observations agree. Read the NetBird
route back: one peer, `0.0.0.0/0`, `skip_auto_apply=true`, intended group only.
Prove A forwards and B cannot using the same healthy endpoint.

Rename using PATCH, then regroup through the contract fields. Check native
display/route update and both groups' real traffic; don't infer display name
from stable machine labels. Disable with fresh ETag and key. The selected A
exit must lose forwarding. Reconnect its agent through the documented hook,
then `docker restart fvpn-w4-members-node`. Re-read intent, observed route,
generation/ACK, operation history and flow denial after each. Intent stays
disabled and policy never reopens from heartbeat or startup. Preserve the
pending/unconfirmed phase in UI evidence rather than sampling only final state.

## AT09 — switching, NAT, relay, MTU (§7.2, §8.3)

Two separate node/exit identities are required. Publish both and read back
`skip_auto_apply=true`. There is no central POST exit-selection route in the
44-endpoint contract. Server API manages publication; selection is a native
client operation. The pinned lab CLI already supports:

```sh
docker exec fvpn-w4-members-member netbird routes list
# Set these from this stack's route readback, never guessed IDs.
docker exec fvpn-w4-members-member netbird routes deselect all
docker exec fvpn-w4-members-member netbird routes select "$AT_EXIT_A_NETWORK_ID"
docker exec fvpn-w4-members-member netbird routes select "$AT_EXIT_B_NETWORK_ID"
```

After each selection compare selected route and controlled echo egress. Stop
exit A only while selected, restore it in cleanup, and record error/recovery
and any original-network fallback; don't promise fastest selection or a client
kill switch. For each A/B and direct/forced-relay pair, record NetBird's observed
connection type, underlay/NAT arrangement and controlled TCP/UDP/QUIC results.
Baseline fixed-duration throughput and loss, then test payload sizes
1200/1280/1360/1400/1472 with IPv4 DF/IPv6 packet-too-big evidence as appropriate.
Record largest successful size, ICMP PMTU signal, TCP MSS, retransmits and QUIC
handshake/recovery; do not install an unmeasured global MTU. Fixture commands
must be supplied by the lab owner; no third-party speed-test endpoint.
Physical second NAT and the four clients are needs operator run.

## AT20 — DNS semantics (§12.4–§12.5, §8.1)

Use controlled zones only. Publish immutable rules via the existing policy and
domain-rule API, record normalized fields and policy hash/revision, wait for
both filtering resolvers to ACK. Repeat queries over UDP and TCP and through
the member system resolver. Required matrix:

| Rule/input | Probe | Expected |
|---|---|---|
| `ExAmPlE.com.` block, subdomains true | `example.com`, `a.example.com` | Canonical `example.com`; both blocked. |
| same rule | `notexample.com` | Ordinary controlled answer; label boundary preserved. |
| exact `exact.example.com`, subdomains false | apex / child | Apex blocked; child follows its own rules. |
| `bücher.example.com.` | Unicode and `xn--bcher-kva.example.com` | Same canonical IDNA rule and result. |
| forced block plus overlapping allow | matching name | Forced block wins; same-layer duplicate/conflict drafts reject. |
| threat-feed hit plus explicit allow | matching name | Allow overrides general feed, preserves fixed isolation. |
| scheme/path/port/query/fragment/user-info/control character | rule create | Validation failure, no revision or active policy change. |
| CNAME alias to blocked target | A/AAAA/CNAME | Record chain/rcode for actual backend; do not infer response filtering. |
| every blocked/allowed name | A/AAAA/HTTPS/SVCB | No equivalent address/hint bypass; record wire response and TTL. |

Default block response is NXDOMAIN; use explicit TEST ONLY short TTL. Query
before change, after activation/cache clear and after the actual observed TTL
expires. Record client cache separately; do not claim DNS changes kill existing
HTTPS connections. Rebind an allowlisted name from the controlled public fixture
to private, metadata, overlay and public-prefix internal fixtures. Resolve
successfully but prove member connections are denied at the IP layer, with a
healthy allowed endpoint as control. No public DNS queries or upstream bypass.

## AT21 — resolver and policy-sync failures (§12.2, §12.5–§12.6)

Record member nameserver configuration and both resolver hashes before faults.
For each fault: query a fresh synthetic name through normal system resolution,
record the destination resolver via scoped counters, outcome/latency, and verify
public fallback absence. Stop resolver A, restore A; stop B, restore B; stop
both, then restore. Exactly one surviving controlled resolver should work;
both unavailable must error or stop resolution, according to declared policy.

Block policy synchronization to one resolver using the owner's fault hook.
Activate a changed draft. That resolver retains last-good; other resolver can
apply; UI reports partial, never all applied. Remove the fault and verify matching
hash/ACK. Repeat with both sync paths broken; no empty/public configuration may
replace last-good. Run Private DNS/DoH script and guidance in OPERATORS.md.

## AT22 — privacy, retention, feeds (§12.3, §12.7, §16.4)

Use a unique synthetic qname and URL path on the controlled endpoint that are
not part of any configured rule, feed or health probe. Generate successful,
blocked and failed queries/requests. Maintain an inventory of all services and
storage locations to avoid calling an empty file scan a pass.

1. Read back AdGuard querylog/statistics configuration and APIs on both
   resolvers; require disabled and no per-domain/per-client history. Inspect
   files for persistence across restart, including work data. Short-lived
   in-memory cache is recorded separately.
2. Scan resolver, proxy, control, worker, agent, origin and payment logs; metrics
   label values; audit before/after diffs; usage/inbox metadata; and an isolated
   logical backup for the unconfigured sentinel qname/URL. Record per-source
   file/row counts and match counts only. Configured domain rules are declared
   admin history, not member query history; do not simply delete all names.
3. Seed old/young retention fixtures through the owner-approved isolated lab
   path, with complete usage projection receipts and an ineligible gap row.
   Run `purge_retained` for raw usage, rollups and audit, using cutoffs derived
   from current retention settings (recommended 35 days, 13 months, 180 days).
   Assert old eligible rows removed, young/incomplete/gap rows retained, purge
   audit appended, repeat returns zero eligible deletions. Verify too-recent
   cutoff fails. Do not purge billing or use DELETE to bypass append-only guards.
4. Record service-log 7-day rotation, configured backup/WAL retention and a
   restore into a separate isolated DB with deletion policy reapplied. A zero
   row purge on an empty database cannot pass retention. DB admin is not reset.
5. Inventory each enabled feed's source ID, version/hash, approved commercial
   use/redistribution/attribution and success time. Pending approval blocks
   production use. Fetch a controlled good snapshot, then timeout, malformed,
   empty and oversized snapshots; previous hash/rule count/answers stay live
   with explicit error/last-good status. No enabled feed means this step is
   untested, not passed. Do not fetch third-party feeds during this test.

## Cleanup

Restore every injected container/network fault before final teardown. Execute
the assigned `e2e-down` even after a partial startup. Check exact prefix names
for containers, networks, volumes and images; evidence contains only remaining
counts. Delete private test scratch after exit. No other task's stack is touched.

## Follow-up 7: D-077 / D-081 restore re-authorization

1. Admit two native devices for a finite-grant member and establish counted HTTPS. Insert exactly one migration-00188 `device_authorization_fences` row through the assigned lab Postgres; wait for the running worker to observe withdrawal.
2. Record the full member list with explicit `true` on the fenced device and `false` on siblings. Traverse the admin list twice with limit 1 and its opaque cursor, checking stable ascending `(member_id, device_id)`, `member_id`, explicit flags and `authorization_unconfirmed_since` timestamp/null. Reusing a cursor with different filters must return 400 `MALFORMED_REQUEST`.
3. Probe the same native HTTPS path while fenced and record route, `fvmb0` arrival, named counters and authenticated origin receipts. A response or counted transfer is a wire-members finding; do not call a missing origin-control response a successful denial.
4. Exercise admin POST with member-only session (401 `UNAUTHENTICATED`), omitted CSRF (403), wrong member (404 `RESOURCE_NOT_FOUND`) and revoked device (409 `STATE_CONFLICT`). Use no request body, the normal Origin and a fresh UUID idempotency key for each independent call. Preserve the fence on denied mutations.
5. Explicit admin authorization must return 202 `member.device_authorize` and remove the fence in the transaction. For an already-authorized fenced device, compare with its existing operation and pre-bump generation; poll the device list after worker sync, never a new operation row. Require `authorized`, explicit `false`, and fresh counted traffic. Replay the same key and require unchanged operation/device version.
6. Produce a pending device through the real two-slot plan downgrade fixture, add its restore fence, and require **409 DEVICE_LIMIT_REACHED** with the fence retained (D-081). Restore the original finite grant; verify that alone does not clear the fence. Re-authorize through HTTP and confirm counted forwarding.
7. Repeat the fence/withdraw/re-grant/traffic cycle on the paid member after authoritative billing reconciliation. Keep payment events, entitlement sources and device confirmation distinct.

The exported `ReplaceFiniteGrant` hook from devstack #284 retires live grants through the production domain command, creates/replays via HTTP, and waits for the worker. Full wire responses, operation identities and fence counts are indexed in [D077.md](D077.md). This is not a PITR drill or a rendered portal-state test. Preserve every failed attempt; never relabel a primary failure as blocked during later readback.


## Follow-up 8 native fixtures

Prepare with `FVPN_E2E_TWO_RESOLVERS=true`. Before enrollment, call
`PrepareNativeUnderlay`, `InstallFixtureProbe`, and `StartFixtureTargets`.
AT09 records bounded 64,064-byte transfers and both admission counters for
IPv4, IPv6 and dual underlay, plus MTU 1500 and 1280. Chosen transport families
are observations; IPv6 overlay is not inferred. QUIC and all four non-Linux
selectors produce explicit blocked rows.

AT02 uses `ReplaceFiniteGrant` for the immediate 3→2 plan transition and
restoration, records credential-free HTTP bodies and operations, and verifies
single audit events under same-key replay. A separate finite deadline expires
without a fallback plan; its actual device counts are recorded. Persistent
TCP/9446 and authenticated UDP/9445 remain active beyond the withdrawal target.
Counter removal and frozen counters are distinguished. The sampling interval
bounds observed reply time; it is not a packet-capture timestamp.

AT20 uses `ConfigureNativeDNS` and `ProbeNativeDNS` for 28 TCP/UDP RR rows.
The pinned convention is direct blocked CNAME/SVCB → empty NODATA and the other
blocked queries → NXDOMAIN. Every row retains RCODE, answer types and SOA TTL.
The DNS owners still need to confirm wording. Authority-only phase changes
measure real positive/negative cache expiry; private A/AAAA connections require
working node listeners and failing member controls. AT21 stops the primary
resolver transport, then both, and measures normal system lookup and recovery.
The exact resolver list comes from product desired state; the peer-restricted
NetBird nameserver group is TEST ONLY distribution plumbing.

## Follow-up 15 evidence and regression readback

Run the foreground gates in RUN.md. `assembled.py` enforces admission before
startup and after startup, runs the exact ten-case target, retains private
producer exports and tears down the owned stack. Record `vmstat 5 7`,
`pgrep -af 'gates.sh|mk/at-'` and `go env GOCACHE` before starting.

New files larger than 20,000,000 bytes use lossless `.gz` and a
`.gz.sha256.json` sidecar with compressed/uncompressed byte counts and SHA-256.
The Go `readEvidence` and Python `evidence_io.load` readers accept a logical
plain filename and validate the compressed form when present. Historical
archives remain byte-identical. To inspect a large artifact directly, use
`gzip -dc <artifact>.gz`; do not replace the retained file.

At the existing AT02 service-stop point, the TEST ONLY probe requests the
supervisor pause and SIGTERM for the retained agent PID, observes process exit,
and joins its parent wait receipt by launch ID/PID/start ticks. The measured
interval starts before either the helper or supervisor can signal: it is an
upper bound, not a kernel exit instant. The existing Service stop/start follows
and retains the selected release and stored lease. Docker restart/namespace
command durations remain distinct from process shutdown intervals. Control I/O
after Step cancellation is not observable on this stream.

Central hold readbacks retain all incident/report/lease/audit IDs and cumulative
history at case end, plus member-state snapshots around original AT02
revocations. Original reports are replayed through the central publication
classifier; these replays are labeled separately from observed operation
transitions. Missing sole-member, sibling-member or retained-safety scenarios
are `not exercised`, not successful tests. No hold or alarm is manufactured.

Every D-077 re-grant keeps its original readiness and existing member/node/origin
packet and counter evidence. A missing launched transfer stays empty rather
than borrowing a previous attempt. `cause_status` remains `unproven`.
Withdrawal parser failures retain mode-0600 private raw/error files. Netlab
23's typed duplicate-chain rejection is accepted as fail-closed policy; no
retry, counter substitution or relaxed predicate is allowed.

After the run, execute `followup15_index.py`, `followup15_evidence.py`,
`followup15_receipts.py` and `followup15_transitions.py` with `python3 -B` from
the repository root. They read the original records/private captures and write
public diagnostic indexes; they neither send traffic nor replace live verdicts.
Compress any new oversized derived files and regenerate that new run's pins.
