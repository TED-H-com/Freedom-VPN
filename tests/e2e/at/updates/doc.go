// Package updates is the Wave 4 acceptance driver for releases, node updates
// and rollback (Project Spec §13, §19 AT23–AT27).
//
// Live cases use the build tag e2e and the dev stack in deploy/devstack.
// go test without that tag runs the evidence helpers only. The lab helpers in
// tests/e2e are unexported, so this package talks to the stack through
// stack.json, the admin API, docker exec and the lab database.
package updates
