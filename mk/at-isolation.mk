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

.PHONY: at-isolation-lint at-isolation at-isolation-review-captures at-isolation-lab-up at-isolation-lab-down

at-isolation-disk = python3 tests/e2e/at/isolation/run.py

at-isolation-lint:
	@test -z "$$(gofmt -l tests/e2e/at/isolation tests/netlab/at tests/e2e/at/isolation/driver.go.txt)" || { echo 'gofmt required'; exit 1; }
	@$(at-isolation-disk) go vet $(AT_PACKAGES)
	@$(at-isolation-disk) go vet -tags netlab ./tests/netlab/at/...
	@$(at-isolation-disk) go run $(AT_STATICCHECK) $(AT_PACKAGES)
	@$(at-isolation-disk) go run $(AT_STATICCHECK) -tags netlab ./tests/netlab/at/...
	@$(at-isolation-disk) go vet tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
	@$(at-isolation-disk) go run $(AT_STATICCHECK) tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go
	@python3 tests/e2e/at/isolation/driver.py lint

# Offline evaluation only; archive the original AT13 first (see RUN.md).
at-isolation-review-captures:
	@FVPN_AT_REVIEW_DIR='$(abspath $(FVPN_AT_EVIDENCE_DIR))' $(at-isolation-disk) go test -count=1 -v -run '^TestAT13_ReviewCapturedExpiry$$' ./tests/e2e/at/isolation

at-isolation-lab-up:
	FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' sh tests/e2e/at/isolation/lab.sh up

at-isolation-lab-down:
	FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' sh tests/e2e/at/isolation/lab.sh down

at-isolation-test:
	@$(at-isolation-disk) go test -count=1 $(AT_PACKAGES)

at-isolation-build:
	@$(at-isolation-disk) python3 tests/e2e/at/isolation/driver.py build
	@$(at-isolation-disk) sh -ec 'for arch in amd64 arm64; do GOOS=linux GOARCH=$$arch go build -o "$$GOTMPDIR/probe-$$arch" tests/e2e/at/isolation/probe_main.go tests/e2e/at/isolation/observer_main.go; done'

at-isolation:
	@$(at-isolation-disk) python3 tests/e2e/at/isolation/run.py assembled
