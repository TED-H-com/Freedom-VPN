# Members requests and external blockers

Current results are in [summary.md](summary.md). Earlier request text and its final-run evidence are preserved in the [follow-up 7 archive](history/follow-up-7/requests.md).

## Follow-up 8 — 2026-10-31: fixture adoption and remaining scope

Baseline `6848add` includes devstack follow-up 16 (#284). The new harness uses
`ReplaceFiniteGrant`, the native underlay/MTU selectors, and native two-resolver
DNS/RR/cache/rebinding hooks. Follow-up 15 (#266) supplies persistent member
TCP/9446 and authenticated UDP/9445 plus forced relay. Current outcomes belong
to this follow-up's root case records; the final follow-up 7 run is now preserved
byte-identically in [history/follow-up-7](history/follow-up-7/RUN.md), with its
run ID and checksum manifest (D-098).

### Finite expiry versus immediate plan replacement

Owner: **w3-wire-members / entitlement owner**, with devstack for fixture scope.
`ReplaceFiniteGrant` immediately revokes the old finite grant and creates the
replacement. It is not a scheduled future downgrade. The grant-only member's
real deadline is tested separately, without fabricating a lower subscription
or overlapping different-plan grants. Confirm whether the requested deadline
transition means expiry or a new scheduled plan-change contract. Exact HTTP
responses, operation IDs, revisions, device counts, audit cardinality and
persistent transport samples are in AT02.json.

### F4-03 / F4-04

Owner: **w3-devstack-e2e, platform/operator campaign, and portal for screens**.
#266 closes the missing persistent member UDP and forced-relay fixture exports;
#284 closes the missing native IPv4/IPv6/dual and finite MTU input exports.
This run measures the consuming AT rows. QUIC and Android/iOS/Windows/macOS
selectors explicitly declare unavailable; physical second NAT/roaming and
ICMP-black-hole PMTU remain open. Rendered suspended/expired member screens
remain open; API state cannot supply those screenshots/assertions.

### F4-05 / F4-07

Owner: **w3-devstack-e2e / native integration and DNS/netlab owners**.
#284 supplies native DNS setup, two-resolver fallback, controlled RR types,
cache phases and private A/AAAA targets. The member runner consumes these
fixtures and retains RCODE, SOA TTL and answer types per TCP/UDP case. Direct
blocked CNAME/SVCB NODATA must not be described as NXDOMAIN. Confirm §12's
wording for that backend behavior. A positive IDNA authority record is still
needed for a wire-level normalization assertion. Product automatic nameserver
distribution and Private DNS/DoH platform runs remain unclaimed; fixture
nameserver distribution is expressly TEST ONLY.

### F4-06 / F4-10

Owners remain **devstack/usage/database/backup** (F4-06) and
**packaging/devstack/operator campaign** (F4-10). Raw-usage/rollup/file/backup
retention, representative URL privacy and commercial feed approval are not
closed by count-only transport/DNS probes. #279 fixes installer startup and
seed lifecycle; #282 adds enrollment evidence. Neither supplies all three
AT05 source flavours or physical arm64 to this lane. The member baseline
continues to report that boundary explicitly.

### F4-08 / F4-09 ledger definitions

Request to orchestrator: neither identifier is defined in the repository's
member request/blocker records or their tracked history. Please supply the
original definitions and owners; no closure or substitute owner is inferred.
The separately documented worker RestartPolicy issue was closed by #240 and
is freshly rechecked in AT05.

### F8-01 — second-exit namespace restart does not regain serving

Owners: **w2-agent-core and w3-devstack-e2e**, with **w3-wire-nodes** for central
lease/readback coordination. Diagnostic run `942624c2-babc-4610-aa43-9560a9e70683`
on product baseline `6848add`, harness `2259947`, restarts the same node2 identity
after stopping node2, its resolver and namespace holder. The post-restart
observation reports `TC_FOREIGN_FILTER`, `fvmb0` qdisc `read_failed`, and initial
quarantine `down_verify_failed`. Same desired/applied generation `23` remains;
serving does not recover within 90 seconds, and provisional leases keep zero
member grants. AT08's primary-node-only restart passed separately.

The shared `deploy/devstack/entrypoint-node.sh` launches the agent supervisor
and NetBird startup concurrently. The observation is consistent with the
retained tc check running before `fvmb0` returns; this is a source-supported
inference, not a proven unique root cause. Relevant product paths are
`node/agent/engine.go`, `node/enforce/verify_tc.go` and
`node/enforce/quarantine.go`. Please provide a bounded supported restart that
preserves fail-closed containment and eventually regains serving after the
native link is ready. No state clearing, fabricated ACK, forced generation or
agent repair was used in this member case. The final rerun retains the same
assertion; its result is authoritative for reproducibility.

### Final run result — F8-01 reproduced

Authoritative run `69f92f49-73dc-4779-8101-dfb6a802a96b`, harness
`d79accd9c693aed83713072c925b3b5005b5684c`, same product baseline `6848add`.
The second namespace restarted at `2026-10-31T09:21:57.624783604Z`; desired and
applied generation stayed `30`, but serving did not return within 90 seconds
and there was no first counted byte. `TC_FOREIGN_FILTER`,
`AWAITING_MEMBER_GRANTS` and `HEALTH_CHECK_FAILED` recur in the retained
observations. AT08's primary node-only restart passed. Three AT09 restart
assertions fail; they are not demoted to blocked. The namespace fault ran after
AT20–AT22, whose executed checks all passed.

### F8-02 — MTU 1280 bulk transfer incomplete

Owners: **w3-devstack-e2e and native transport/netlab**, with **w3-wire-nodes**
for enforcement-path diagnosis. In the same final run, `ApplyNativeInputs`
accepted dual underlay / MTU 1280; the member was taken down/up and its selected
exit was connected via direct IPv4 before and after the bounded transfer.
The fixture received only **11,011 / 64,064 bytes** after **12,011 ms**, with
`bulk_echo_incomplete`. Device upload/download increased by **66,296 / 45,963
bytes**. The earlier three 1500 rows returned all 64,064 bytes (IPv4 → IPv4,
IPv6 → IPv6 and dual → IPv6). Exact input, path and counters are in AT09.json
and [NATIVE.md](NATIVE.md).

Please determine the MTU/MSS/echo-path cause and provide a repeatable supported
fixture. A completed transfer in the diagnostic attempt or devstack's own run
does not pass this failed row. No retry, counter reset or product change was used to turn this row into a
pass. The combined members matrix reconnects NetBird before each family/MTU
row; devstack's standalone MTU scenario retains the existing native session.
This difference and the observed IPv6-to-IPv4 path choice are reproduction
inputs, not a proven root cause. The following independent relay row restores
MTU 1500 first. Forced relay (two relayed, zero
direct peers), direct restoration (zero relayed, two direct peers), and TCP/UDP
continuity on the same sockets subsequently passed.
