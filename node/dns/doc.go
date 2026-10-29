// Package dns converges a canonical resolver document onto AdGuard Home and
// returns the evidence the agent puts in its observation report (§12.5, §17.1).
//
// The agent reports checks[name=dns_policy].status = passed only when
// Evidence.Verified is true. That check is the dns_ok advertisement: there is
// no separate dns_ok field on the observation check. When Verified is false the
// node must not advertise dns_ok and must not claim the policy generation in
// applied_policy_refs.
//
// This package does not edit nftables, does not meter member traffic, and does
// not emit usage events. DNS service counters stay dns_service_v1 in
// node/enforce (§8.2, §10.2).
package dns
