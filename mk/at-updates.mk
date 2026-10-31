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
# Reuse a healthy, freshly prepared owned stack (before its first enrollment).
AT_UPDATES_REUSE ?= 0

export GOTOOLCHAIN := local
export CGO_ENABLED := 0
export AT_UPDATES_REUSE

LINT_TARGETS += at-updates-lint

.PHONY: at-updates-lint at-updates

at-updates-lint:
	@test -z "$$(gofmt -l tests/e2e/at/updates)" || { echo 'gofmt required'; exit 1; }
	@python3 tests/e2e/at/updates/run.py lint

# Check shared-host admission before invoking this foreground target.
# Records precede the gate; Python owns teardown even when a check fails.
at-updates:
	@python3 tests/e2e/at/updates/run.py live
