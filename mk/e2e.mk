# Dev-stack acceptance targets. e2e-test is privileged and is not in TEST_TARGETS.
# e2e-lint joins the root lint aggregate.
#
#   make -f mk/e2e.mk e2e-up
#   make -f mk/e2e.mk e2e-test
#   make -f mk/e2e.mk e2e-down
#
# FVPN_E2E_PREFIX (default e2e) names runtime objects fvpn-<prefix>-*.
# Shared build-cache images use neutral fvpn-devstack-* names.
# FVPN_E2E_PORT_BASE (default 18000) is added to a fixed offset for each
# loopback port. Legal bases are 6000+6000*n, n=0..9; existing slots stay fixed.
# This task uses only prefix e2e and base 18000. Bases 24000, 30000, 36000,
# Other slot owners are listed in docs/e2e/RUN.md.
# The state directory is .agent/e2e-state-<prefix>.

E2E_MK := $(lastword $(MAKEFILE_LIST))
E2E_ROOT := $(abspath $(dir $(E2E_MK))/..)
FVPN_E2E_PREFIX ?= e2e
FVPN_E2E_PORT_BASE ?= 18000
FVPN_E2E_STATE ?= $(E2E_ROOT)/.agent/e2e-state-$(FVPN_E2E_PREFIX)
export FVPN_E2E_PREFIX
export FVPN_E2E_PORT_BASE
export FVPN_E2E_STATE
export GOTOOLCHAIN := local
export CGO_ENABLED := 0

LINT_TARGETS += e2e-lint

.PHONY: e2e-lint e2e-config e2e-up e2e-down e2e-test

e2e-lint:
	@test -z "$$(gofmt -l $$(git ls-files --cached --others --exclude-standard 'tests/e2e/**/*.go'))" || { echo 'gofmt required'; exit 1; }
	cd $(E2E_ROOT) && go vet ./tests/e2e/...
	cd $(E2E_ROOT) && go vet -tags e2e ./tests/e2e/...
	cd $(E2E_ROOT) && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/...
	cd $(E2E_ROOT) && go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/...

e2e-config:
	FVPN_E2E_STATE=$(E2E_ROOT)/.agent/e2e-config-placeholder docker compose -p fvpn-e2e -f $(E2E_ROOT)/deploy/devstack/compose.yaml config -q

e2e-up:
	FVPN_E2E_PREFIX=$(FVPN_E2E_PREFIX) FVPN_E2E_PORT_BASE=$(FVPN_E2E_PORT_BASE) FVPN_E2E_STATE=$(FVPN_E2E_STATE) E2E_RESET=$(E2E_RESET) $(E2E_ROOT)/deploy/devstack/up.sh

e2e-down:
	FVPN_E2E_PREFIX=$(FVPN_E2E_PREFIX) FVPN_E2E_PORT_BASE=$(FVPN_E2E_PORT_BASE) FVPN_E2E_STATE=$(FVPN_E2E_STATE) $(E2E_ROOT)/deploy/devstack/down.sh

# Fresh secrets, then the acceptance run. The stack stays up for inspection.
e2e-test:
	$(MAKE) -f $(E2E_MK) e2e-down
	$(MAKE) -f $(E2E_MK) e2e-up E2E_RESET=1
	cd $(E2E_ROOT) && FVPN_E2E=1 FVPN_E2E_PREFIX=$(FVPN_E2E_PREFIX) FVPN_E2E_PORT_BASE=$(FVPN_E2E_PORT_BASE) FVPN_E2E_STATE=$(FVPN_E2E_STATE) go test -v -tags e2e -count=1 -timeout 65m ./tests/e2e
