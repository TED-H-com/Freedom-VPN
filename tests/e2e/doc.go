// Package e2e is the dev-stack acceptance runner (§19).
// Files without an e2e build tag stay free of Docker. The scenario test is
// built with -tags e2e and runs only when FVPN_E2E=1.
package e2e
