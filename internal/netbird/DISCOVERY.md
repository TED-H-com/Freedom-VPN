# Enrollment peer discovery (D-055)

`CreateNodeSetupKey(ctx, enrollmentID)` uses the enrollment operation UUID as its
stable attempt ID. Alongside `fvpn-unpublished-nodes` it creates an empty group
named `stableName("enroll-peer", enrollmentID)` and stores that group under
`enroll-peer:<enrollmentID>`. The one-off setup key has exactly these two
auto-groups, usage limit 1, external lifetime 86400 seconds and `ephemeral=false`.
Creation confirms settings with GET; a lost creation response never issues a
second key or recovers plaintext. Existing membership is preserved on retry.

`DiscoverNodePeer(ctx, enrollmentID, NodePeer{NodeID, IncarnationID})` takes the
same deployment lock as every adapter mutation. It locates the attempt's key and
group, reads both, checks key settings/usage/revocation and checks all policies, routes and
DNS nameserver groups for references (including `authorized_groups` object keys). Zero peers is `NotFound`. Exactly one peer with
`used_times=1`, `user_id=""` and `ephemeral=false` supplies the management proof.
Multiple peers, usage above one, mismatching group settings/bindings, user-owned
or ephemeral peers, revoked keys and policy/route references return `Conflict`.
Malformed/incomplete read-back is `Unconfirmed`. `DiscoveryReason` exposes only
fixed diagnostic strings, never upstream bodies, keys, tokens or addresses.

The adapter stores the same `peers` / `node:<node>:<incarnation>` binding as
`EnsureNodePeer`. It then removes the temporary group from the exhausted key,
confirms that update, empties the group with read-back and deletes it with a
confirming GET. The shared staging group and its membership remain. A separate `enrollment-withdrawal` receipt retains the confirmed peer for
withdrawal even if central commit is lost. Binding before cleanup makes it replayable; a second discovery returns
the recorded peer and completes any remaining cleanup without duplicate resources.
`RevokeNodeSetupKey` confirms revocation, re-reads the key/group, and deletes a
sole machine peer proven by one key use, even before `recordPeer`. It requires
GET 404 before deleting the group. A retained withdrawal receipt also allows
revocation after discovery removed the group but central commit was lost. Zero
members is a no-op; ambiguous membership or unsafe ownership is a conflict, with
the group retained. A missing key with a remaining group is not evidence of
withdrawal. Known node bindings are removed separately after route withdrawal.
A confirmed `enrollment-cleanup` marker permits retry after detaching an unused
revoked key; it does not authorize a peer mapping.

The worker supplies `WithDiscoveryFence`: short DB checks after taking the adapter
lock and immediately before binding validate its claim and current unrevoked,
unfrozen incarnation. These checks hold no transaction across HTTP. The worker
also fences its final projection/evidence transaction; withdrawal uses the same
deployment lock and removes an interrupted discovery's adapter binding even if
the central `peer_id` was not yet committed.

NetBird 0.82.0 considers a one-off key `overused` at its first use; this is accepted
only with `used_times=1`. Expiry after registration does not invalidate the peer;
explicit revocation always rejects discovery. These are management read-back
rules, not publication or member-admission decisions. Native client/real-network
behavior still requires the lab; local HTTP fakes are **TEST ONLY**.

Run `go test -count=1 ./internal/netbird` for key creation, success/replay,
conflicts, cleanup retry, revocation and claim-fence adapter tests. Central tests
are `go test -tags integration ./apps/worker -run D055` with an admin
`FVPN_TEST_DATABASE_URL`; each test creates/drops its own isolated database.

D-055 adversarial fixes: re-GET both key and group after reading the candidate
and again inside cleanup, including after detaching the key. A 0-use snapshot
with one member is retryable `Unconfirmed`; a subsequent 1-use read can bind.
A second use/member conflicts, and a binding rejected during cleanup is removed.
This includes a binding left by a previous delivery whose cleanup failed transiently;
a later conflict must remove it before terminal worker acknowledgement.
The separately retained withdrawal target is never a serving identity. Compare
optional `peers_count`/`resources_count` against complete arrays; disagreement is
`Unconfirmed`, and nonempty resources conflict. Membership is checked before
emptying and again before DELETE; every mutation requires read-back. Dashboard
writes remain excluded from adapter-owned objects because management has no CAS.

A legacy key lacking exactly the two creation groups, or a used key whose
correlation disappeared without durable cleanup proof, returns the fixed reason
`correlation_unavailable_revoke_and_re_enroll`. An operator must revoke this
attempt and re-enroll; no hostname/IP/time fallback exists. `DiscoveryReason`
allowlists all public reasons; unknown error text is never placed in audit data.
