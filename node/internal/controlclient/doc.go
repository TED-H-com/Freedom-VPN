// Package controlclient is the exit node's mTLS client for the agent API
// (§14.6, §14.7, D-015).
//
// Lease renewal, usage upload and every other call use separate connection
// pools and concurrency budgets, so a large usage batch cannot occupy the
// sockets a lease renewal needs. The client pins the control CA, refuses
// redirects, and caps each body at canonjson.MaxBytes. It does not log
// tokens, setup keys or request bodies.
package controlclient
