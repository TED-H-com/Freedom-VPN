// Package fakecontrol is an in-process agent API for tests (TEST ONLY).
//
// It implements enrollment, certificate renewal, desired state, observations,
// lease renewal, NetBird bootstrap, usage batches and the signed usage
// checkpoint. Signing keys are generated at runtime and are not product keys.
// The server can inject 5xx responses, delays, bad signatures, old generations,
// a revoked identity, clock skew, and usage 503, 429 and checkpoint watermarks.
package fakecontrol
