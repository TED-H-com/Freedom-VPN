// Package fakecontrol is an in-process agent API for tests (TEST ONLY).
//
// It implements enrollment, certificate renewal, desired state, observations,
// lease renewal and NetBird bootstrap. Signing keys are generated at runtime
// and are not product keys. The server can inject 5xx responses, delays,
// bad signatures, old generations, a revoked identity and clock skew.
package fakecontrol
