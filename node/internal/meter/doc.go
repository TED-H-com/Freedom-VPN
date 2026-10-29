// Package meter turns kernel admission counters into the usage event stream
// (§10.1, §10.5, §10.7, §10.9, §13.6 steps 8 and 11).
//
// Usage is the inner IPv4 total length admitted at the forward hook, read
// from the node/enforce named counters (decision D-019). Bytes later dropped
// by HTB are not subtracted. DNS to the resolver (dns_service_v1) and
// diagnostic counters are not member usage.
//
// A counter that disappears or goes backwards, including a uint64 wrap, is
// enforce.DetectResets reason METER_EPOCH_CHANGE. That opens a new meter
// epoch and a gap. It is never a negative delta. event_seq is not reset.
//
// The updater calls Barrier and then FinalSample after member forwarding is
// blocked (§13.6 step 8). Health tells the forwarding gate when the ledger
// cannot be persisted or an outbox cap is reached (§10.7, §10.8). This
// package does not install nftables rules; the agent owns that gate.
package meter
