// Package enforce renders and checks the node data-plane policy.
//
// The agent (a later wave) feeds a typed model. This package turns it into one
// atomic nftables transaction for table inet fvpn and a tc batch that updates
// HTB classes in place. It does not flush the host ruleset, write fwmark, or
// install a flowtable. Verification reads nft -j and tc -j and returns a
// stable reason when the kernel state does not match the model.
//
// Metering is forward_admission_v1 (§10.1, §10.2): one named counter per
// device per direction, incremented only after the lease and isolation checks,
// on the inner IP packet, before SNAT. DNS-to-resolver and diagnostic drops
// use other counter names and are not usage.
package enforce
