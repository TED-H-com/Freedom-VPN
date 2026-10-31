# AT14–AT19 (§19): usage metering, ledger and Fair-Use acceptance targets.
# Assigned stack instance: FVPN_E2E_PREFIX=w4-usage FVPN_E2E_PORT_BASE=48000.

export GOTOOLCHAIN := local
export CGO_ENABLED := 0

FVPN_E2E_PREFIX ?= w4-usage
FVPN_E2E_PORT_BASE ?= 48000
AT_USAGE_STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.8.2
AT_USAGE_PKGS := ./tests/e2e/at/usage/...

at-usage-disk = set -eu; mkdir -p .agent/tmp; at_tmp="$$(pwd)/.agent/tmp"; export TMPDIR="$$at_tmp" GOTMPDIR="$$at_tmp";

LINT_TARGETS += at-usage-lint
TEST_TARGETS += at-usage-test
BUILD_TARGETS += at-usage-build

.PHONY: at-usage at-usage-test at-usage-lint at-usage-build

at-usage:
	@$(at-usage-disk) FVPN_E2E=1 FVPN_E2E_PREFIX='$(FVPN_E2E_PREFIX)' FVPN_E2E_PORT_BASE='$(FVPN_E2E_PORT_BASE)' \
		go test -tags e2e -count=1 -p 1 -parallel 2 -timeout 30m -v $(AT_USAGE_PKGS)

at-usage-test:
	@$(at-usage-disk) go test -p 1 -parallel 2 $(AT_USAGE_PKGS)

at-usage-lint:
	@test -z "$$(gofmt -l tests/e2e/at/usage)" || { echo 'gofmt required in tests/e2e/at/usage'; gofmt -l tests/e2e/at/usage; exit 1; }
	@$(at-usage-disk) go vet $(AT_USAGE_PKGS)
	@$(at-usage-disk) go vet -tags e2e $(AT_USAGE_PKGS)
	@$(at-usage-disk) go run $(AT_USAGE_STATICCHECK) $(AT_USAGE_PKGS)
	@$(at-usage-disk) go run $(AT_USAGE_STATICCHECK) -tags e2e $(AT_USAGE_PKGS)

at-usage-build:
	@$(at-usage-disk) for arch in amd64 arm64; do \
		GOOS=linux GOARCH=$$arch go test -c -tags e2e -o "$$at_tmp/at-usage-$$arch.test" ./tests/e2e/at/usage; \
	done; rm -f "$$at_tmp/at-usage-amd64.test" "$$at_tmp/at-usage-arm64.test"
