# Wave 4 acceptance: releases, node updates and rollback (AT23–AT27).
# The live target is privileged and is not in TEST_TARGETS.
#
#   make -f mk/at-updates.mk at-updates-lint
#   make -f mk/at-updates.mk at-updates
#
# The live run uses its own stack. Legal port bases are 18000+6000*n.

AT_UPDATES_MK := $(lastword $(MAKEFILE_LIST))
AT_UPDATES_ROOT := $(abspath $(dir $(AT_UPDATES_MK))/..)
AT_UPDATES_PREFIX := w4-updates
AT_UPDATES_PORT_BASE := 42000

export GOTOOLCHAIN := local
export CGO_ENABLED := 0

LINT_TARGETS += at-updates-lint

.PHONY: at-updates-lint at-updates

at-updates-lint:
	@test -z "$$(gofmt -l tests/e2e/at/updates)" || { echo 'gofmt required'; exit 1; }
	cd $(AT_UPDATES_ROOT) && go vet ./tests/e2e/at/updates
	cd $(AT_UPDATES_ROOT) && go vet -tags e2e ./tests/e2e/at/updates
	cd $(AT_UPDATES_ROOT) && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/updates
	cd $(AT_UPDATES_ROOT) && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/at/updates

# Fresh stack, one acceptance run, then tear-down even when the test fails.
# Blocked product results are recorded and do not fail the Go test.
at-updates:
	FVPN_E2E_PREFIX=$(AT_UPDATES_PREFIX) FVPN_E2E_PORT_BASE=$(AT_UPDATES_PORT_BASE) $(MAKE) -f $(AT_UPDATES_ROOT)/mk/e2e.mk e2e-down
	FVPN_E2E_PREFIX=$(AT_UPDATES_PREFIX) FVPN_E2E_PORT_BASE=$(AT_UPDATES_PORT_BASE) E2E_RESET=1 $(MAKE) -f $(AT_UPDATES_ROOT)/mk/e2e.mk e2e-up
	status=0; \
	cd $(AT_UPDATES_ROOT) && FVPN_E2E=1 FVPN_E2E_PREFIX=$(AT_UPDATES_PREFIX) FVPN_E2E_PORT_BASE=$(AT_UPDATES_PORT_BASE) \
		go test -tags e2e -count=1 -timeout 60m -p 1 -parallel 2 ./tests/e2e/at/updates || status=$$?; \
	FVPN_E2E_PREFIX=$(AT_UPDATES_PREFIX) FVPN_E2E_PORT_BASE=$(AT_UPDATES_PORT_BASE) $(MAKE) -f $(AT_UPDATES_ROOT)/mk/e2e.mk e2e-down; \
	exit $$status
