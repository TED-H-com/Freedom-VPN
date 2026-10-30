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
group, reads both, checks key settings/usage/revocation and checks all policies and
routes for group references. Zero peers is `NotFound`. Exactly one peer with
`used_times=1`, `user_id=""` and `ephemeral=false` supplies the management proof.
Multiple peers, usage above one, mismatching group settings/bindings, user-owned
or ephemeral peers, revoked keys and policy/route references return `Conflict`.
Malformed/incomplete read-back is `Unconfirmed`. `DiscoveryReason` exposes only
fixed diagnostic strings, never upstream bodies, keys, tokens or addresses.

The adapter stores the same `peers` / `node:<node>:<incarnation>` binding as
`EnsureNodePeer`. It then removes the temporary group from the exhausted key,
confirms that update, empties the group with read-back and deletes it with a
confirming GET. The shared staging group and its membership remain. Binding
before cleanup makes interrupted cleanup replayable; a second discovery returns
the recorded peer and completes any remaining cleanup without duplicate resources.
`RevokeNodeSetupKey` confirms revocation and performs the same temporary-group
cleanup, including when the setup key is already absent. It does not disconnect
peers; the central withdrawal adapter deletes known peer bindings separately.

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
