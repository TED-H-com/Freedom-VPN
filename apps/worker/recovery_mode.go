package worker

// Recovery permits only work that can make restored state safe or converge it.
var recoveryAllowedTypes = []string{
	"member.entitlement_changed", "member.grant_revoked", "member.suspended",
	"device.authorized", "device.revoked", "netbird.device_sync", "members.reconcile",
	"billing.reconcile", "billing.inbox_received",
	"node.staleness.sweep", "node.generation_hold.check", "node.peer.observe",
	"node.resolver_documents.reconcile", "node.netbird_routes.reconcile",
	"dns.publish", "dns.feed.refresh",
}

const recoveryFilteredType = "node.publication_changed"

var recoveryWithdrawalKinds = []string{
	"node.unpublish", "node.disable", "node.revoke", "node.retire",
}

// These registered types wait until normal mode. Unlisted types also wait.
var recoveryWaitingTypes = []string{
	"devices.discover", "node.reconcile", "fleet.update_created", "fleet.first_sync_created",
	"fleet.update_retry", "fleet.release_support_changed", "node.updated", "node.provision.requested",
	"node.peer.discover", "release.verify", "policy.activated", "fup.evaluate",
	"fup.member_generation_changed", "usage.projection_rebuild", "usage.retention_purge",
}
