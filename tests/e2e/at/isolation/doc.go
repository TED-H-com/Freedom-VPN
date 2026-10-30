// Package isolation is the Wave 4 acceptance runner for AT10–AT13 (§19).
//
// The privileged-lab half lives in tests/netlab/at and writes mechanism
// evidence. The product half, built with -tags e2e, drives the dev stack
// from deploy/devstack through the API and the member container. Neither
// half is a pass for a step it did not observe.
package isolation
