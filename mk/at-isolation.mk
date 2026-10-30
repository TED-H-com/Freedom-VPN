# AT10–AT13 (§19). at-isolation is privileged and is not in TEST_TARGETS.
# at-isolation-lint is the same gofmt, vet and staticcheck pin as mk/go.mk,
# limited to this task's packages.
#
#   make -f mk/at-isolation.mk at-isolation-lint
#   make -f mk/at-isolation.mk at-isolation
#
# The stack binding is prefix w4-isolation and port base 36000
# (control 36443, IdP 36444, origin 36445, Postgres 41432).

AT_MK := $(lastword $(MAKEFILE_LIST))
AT_ROOT := $(abspath $(dir $(AT_MK))/..)
FVPN_E2E_PREFIX ?= w4-isolation
FVPN_E2E_PORT_BASE ?= 36000
FVPN_LAB_PREFIX ?= fvpn-w4-isolation-lab
FVPN_AT_EVIDENCE_DIR ?= docs/e2e/at/isolation
AT_STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.8.2
AT_PACKAGES := ./tests/e2e/at/isolation/... ./tests/netlab/at/...

export GOTOOLCHAIN := local
export CGO_ENABLED := 0

LINT_TARGETS += at-isolation-lint

.PHONY: at-isolation-lint at-isolation at-isolation-lab-up at-isolation-lab-down

at-isolation-disk = set -eu; mkdir -p .agent/tmp; at_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/at-isolation.XXXXXX"); trap 'rm -rf "$$at_tmp"' EXIT; export TMPDIR="$$at_tmp" GOTMPDIR="$$at_tmp";

at-isolation-lint:
	@test -z "$$(gofmt -l tests/e2e/at/isolation tests/netlab/at)" || { echo 'gofmt required'; exit 1; }
	@$(at-isolation-disk) go vet $(AT_PACKAGES)
	@$(at-isolation-disk) go vet -tags e2e ./tests/e2e/at/isolation/...
	@$(at-isolation-disk) go vet -tags netlab ./tests/netlab/at/...
	@$(at-isolation-disk) go run $(AT_STATICCHECK) $(AT_PACKAGES)
	@$(at-isolation-disk) go run $(AT_STATICCHECK) -tags e2e ./tests/e2e/at/isolation/...
	@$(at-isolation-disk) go run $(AT_STATICCHECK) -tags netlab ./tests/netlab/at/...

at-isolation-lab-up:
	FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' sh tests/e2e/at/isolation/lab.sh up

at-isolation-lab-down:
	FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' sh tests/e2e/at/isolation/lab.sh down

# Lab first, then the dev stack. Either half failing still tears both prefixes down.
at-isolation:
	@$(at-isolation-disk) \
	lab_status=0; stack_status=0; \
	trap '$(MAKE) -f mk/at-isolation.mk at-isolation-lab-down FVPN_LAB_PREFIX="$(FVPN_LAB_PREFIX)" || true; $(MAKE) -f mk/e2e.mk e2e-down FVPN_E2E_PREFIX="$(FVPN_E2E_PREFIX)" FVPN_E2E_PORT_BASE="$(FVPN_E2E_PORT_BASE)" || true; rm -rf "$$at_tmp"' EXIT; \
	$(MAKE) -f mk/at-isolation.mk at-isolation-lab-up FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' || lab_status=$$?; \
	if [ $$lab_status -eq 0 ]; then \
		cd $(AT_ROOT) && FVPN_NETLAB=1 FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' FVPN_AT_EVIDENCE_DIR='$(FVPN_AT_EVIDENCE_DIR)' \
			go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 ./tests/netlab/at || lab_status=$$?; \
	fi; \
	$(MAKE) -f mk/at-isolation.mk at-isolation-lab-down FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' || lab_status=$$?; \
	$(MAKE) -f mk/e2e.mk e2e-down FVPN_E2E_PREFIX='$(FVPN_E2E_PREFIX)' FVPN_E2E_PORT_BASE='$(FVPN_E2E_PORT_BASE)' || stack_status=$$?; \
	$(MAKE) -f mk/e2e.mk e2e-up FVPN_E2E_PREFIX='$(FVPN_E2E_PREFIX)' FVPN_E2E_PORT_BASE='$(FVPN_E2E_PORT_BASE)' E2E_RESET=1 || stack_status=$$?; \
	if [ $$stack_status -eq 0 ]; then \
		cd $(AT_ROOT) && FVPN_E2E=1 FVPN_E2E_PREFIX='$(FVPN_E2E_PREFIX)' FVPN_E2E_PORT_BASE='$(FVPN_E2E_PORT_BASE)' FVPN_AT_EVIDENCE_DIR='$(FVPN_AT_EVIDENCE_DIR)' \
			go test -tags e2e -count=1 -timeout 25m -p 1 -parallel 1 ./tests/e2e/at/isolation || stack_status=$$?; \
	fi; \
	$(MAKE) -f mk/e2e.mk e2e-down FVPN_E2E_PREFIX='$(FVPN_E2E_PREFIX)' FVPN_E2E_PORT_BASE='$(FVPN_E2E_PORT_BASE)' || stack_status=$$?; \
	left=$$( { docker ps -a --format '{{.Names}}'; docker images --format '{{.Repository}}:{{.Tag}}'; docker network ls --format '{{.Name}}'; docker volume ls --format '{{.Name}}'; } | grep -E '^fvpn-w4-isolation(-|$$)' || true ); \
	if [ -n "$$left" ]; then echo "prefix objects remain:"; echo "$$left"; exit 1; fi; \
	echo "at-isolation lab=$$lab_status stack=$$stack_status"; \
	if [ $$lab_status -ne 0 ] || [ $$stack_status -ne 0 ]; then exit 2; fi
