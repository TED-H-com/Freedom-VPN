// Package healthattest is the file contract between the node updater and the
// candidate agent for §13.9 isolation, DNS and tunnel checks (D-053).
//
// The updater writes one challenge into the journal directory. The candidate
// agent reads it, runs the checks, and writes an attestation. The updater
// accepts that attestation only when it is bound to the live challenge.
// A config field, an environment variable or a file that was already there
// cannot mark the checks passed.
//
// The package is stdlib plus pkg/api so the agent can import it. It does not
// decide member forwarding. Publication intent in an attestation is evidence
// the agent reports; the updater's serving rules stay in charge of the gate.
package healthattest
