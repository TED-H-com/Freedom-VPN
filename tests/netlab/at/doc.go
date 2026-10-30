// Package at is the Wave 4 privileged-lab acceptance for AT10–AT13 (§19).
//
// These tests run inside the existing Gate G2 container from tests/netlab.
// They prove the enforcement mechanism: where packets die, that a ruleset
// reload does not open an allow window, and that revocation removes
// established flows. They do not claim the assembled dev stack.
package at
