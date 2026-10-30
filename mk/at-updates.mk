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

at-updates-disk-tmp = set -eu; cd $(AT_UPDATES_ROOT); mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/at-updates.XXXXXX"); export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

export GOTOOLCHAIN := local
export CGO_ENABLED := 0

LINT_TARGETS += at-updates-lint

.PHONY: at-updates-lint at-updates

at-updates-lint:
	@test -z "$$(gofmt -l tests/e2e/at/updates)" || { echo 'gofmt required'; exit 1; }
	@$(at-updates-disk-tmp) trap 'rm -rf "$$task_tmp"' EXIT; \
	go vet ./tests/e2e/at/updates; \
	go vet -tags e2e ./tests/e2e/at/updates; \
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/updates; \
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/at/updates; \
	python3 tests/e2e/at/updates/run.py lint; \
	echo 'at-updates-lint: gofmt, vet and staticcheck (offline + e2e) passed'

# Records first; every assigned AT must pass. Tear down even on failure.
at-updates:
	@$(at-updates-disk-tmp) \
	export FVPN_E2E_PREFIX=$(AT_UPDATES_PREFIX) FVPN_E2E_PORT_BASE=$(AT_UPDATES_PORT_BASE); \
	trap 'status=$$?; trap - EXIT; $(MAKE) -f $(AT_UPDATES_ROOT)/mk/e2e.mk e2e-down || status=$$?; rm -rf "$$task_tmp"; exit $$status' EXIT; \
	if [ "$(AT_UPDATES_REUSE)" != 1 ]; then \
		$(MAKE) -f $(AT_UPDATES_ROOT)/mk/e2e.mk e2e-down; \
		E2E_RESET=1 $(MAKE) -f $(AT_UPDATES_ROOT)/mk/e2e.mk e2e-up; \
	fi; \
	FVPN_E2E=1 python3 tests/e2e/at/updates/run.py live
