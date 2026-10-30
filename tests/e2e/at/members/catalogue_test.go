//go:build e2e

package members

type acceptanceCase struct {
	id, title, requirements, expected, next string
	operator                                bool
}

func catalogue() []acceptanceCase {
	return []acceptanceCase{
		{"AT01", "Identity and first use", "APP01 ID01 MEM01", "Two OAuth clients map through verified identity to the correct member/device; selected exit forwards; sign-out, network change and wake recover correctly.", "Shared-issuer native identity is exercised; serving exit and four platform runs remain.", true},
		{"AT02", "Device quota race and isolation", "MEM01 MEM02 MEM03 SEC01", "One final-slot winner, loser denied at gateway, no cross-member disclosure, exactly one slot released by revoke.", "Native quota race, ownership, actual denial, revoke/replacement traffic and accepted health. See AT02-TRACE.md.", false},
		{"AT03", "Payment replay and ordering", "BILL01 BILL03 ENT01", "Ten identical paid events and all six created/paid/failed orders preserve one paid interval; bad signature and forged return grant nothing.", "Paid-period truth, replay/orders, Portal return and counted paid-device forwarding.", false},
		{"AT04", "Expiry, cancellation, refunds and suspension", "BILL02 ENT02 MEM03", "Cancellation preserves paid time; explicit refund policy applies; payment never clears suspension; existing/new TCP and UDP stop on every online node within the configured 60-second target.", "Paid refunds/cancel/expiry/suspension and two-exit TCP refund withdrawal; member UDP and all-event flow timing remain.", false},
		{"AT05", "Three node join sources", "NODE01 NODE04 ENR01 ENR04 PKG01", "Preinstalled, cloud-init and installer nodes use the same protocol, become unpublished, then classify and publish through the same API.", "Three node source flavours; package digests; publication readback; physical arm64 execution.", true},
		{"AT08", "Publication, groups and disable", "NODE02 NODE03 EXIT01 EXIT03", "Only group A forwards; rename/regroup converges; disabled intent and gateway denial survive reconnect and reboot; pending and applied stay distinct.", "D-057 A/B isolation, publication operation history and durable disabled restart, with counted positive controls.", false},
		{"AT09", "Exit selection, NAT, relay and MTU", "EXIT02 APP01 NFR01", "Two exits select correctly with skip_auto_apply, failure behaviour is measured, direct/relay TCP UDP QUIC and PMTU baselines are recorded.", "Two real exits; remaining relay, controlled UDP/QUIC/PMTU fixtures and second NAT/native platform tests.", true},
		{"AT20", "DNS rules and record types", "DNS01 DNS02 ISO04", "Case/dot/IDNA and label boundaries match; URL input rejects; all RR types obey block/TTL; allowlisted rebinding never opens internal networks.", "Controlled RR/rebinding fixtures and actual member DNS configuration; typed resolver delivery is observed separately.", false},
		{"AT21", "DNS failure and encrypted DNS boundary", "DNS03 DNS04 APP01", "Resolver/policy-sync faults keep controlled fallback or fail closed; Android Private DNS and browser DoH limitations match guidance.", "Two filtering resolvers and synchronization fault; native client DNS/fallback plus controlled DoH/DoT and platform operator runs.", true},
		{"AT22", "Privacy, feed sources and retention", "SEC02 DNS01 DNS04", "Synthetic qnames/URLs absent from query history, logs, metrics, audit and backups; bounded retention purges and licensed last-good feed handling verified.", "Candidate configuration, backup/retention fixture and approved feed inventory; queried sentinel traffic and update-failure hook.", false},
	}
}
