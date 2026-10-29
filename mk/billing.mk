# Standalone targets, disk scratch only; same staticcheck pin as mk/go.mk.
export GOTOOLCHAIN := local
export CGO_ENABLED := 0
billing-packages := ./internal/members/... ./internal/entitlement/... ./internal/billing/... ./internal/devices/...
billing-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/billing.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";
BUILD_TARGETS += billing-build
TEST_TARGETS += billing-test
LINT_TARGETS += billing-lint
VERIFY_TARGETS += billing-verify
.PHONY: billing-build billing-test billing-test-integration billing-lint billing-verify
billing-build:
	@$(billing-with-disk-tmp) for arch in amd64 arm64; do GOOS=linux GOARCH=$$arch go build $(billing-packages); done
billing-test:
	@$(billing-with-disk-tmp) go test $(billing-packages)
billing-test-integration:
	@$(billing-with-disk-tmp) go test -tags integration $(billing-packages)
billing-lint:
	@test -z "$$(gofmt -l internal/members internal/entitlement internal/billing internal/devices)" || { echo 'gofmt required'; exit 1; }
	@$(billing-with-disk-tmp) go vet $(billing-packages)
	@$(billing-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 $(billing-packages)
billing-verify: billing-lint billing-test billing-test-integration billing-build
