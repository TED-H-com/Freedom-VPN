# mk/policies.mk: Versioned policies store and fair-use policy (FUP) engine

export GOTOOLCHAIN := local
export CGO_ENABLED := 0

policies-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/policies.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

BUILD_TARGETS += policies-build
TEST_TARGETS += policies-test
LINT_TARGETS += policies-lint
VERIFY_TARGETS += policies-verify

.PHONY: policies-build policies-test policies-test-integration policies-lint policies-verify

policies-build:
	@$(policies-with-disk-tmp) GOOS=linux GOARCH=amd64 go build ./internal/policies/... ./internal/fup/...
	@$(policies-with-disk-tmp) GOOS=linux GOARCH=arm64 go build ./internal/policies/... ./internal/fup/...

policies-test:
	@$(policies-with-disk-tmp) go test -count=1 ./internal/policies/... ./internal/fup/...

policies-test-integration:
	@$(policies-with-disk-tmp) go test -v -count=1 -tags=integration ./internal/policies/... ./internal/fup/...

policies-lint:
	@test -z "$$(gofmt -l internal/policies internal/fup)" || { echo "gofmt needed on: $$(gofmt -l internal/policies internal/fup)"; exit 1; }
	@$(policies-with-disk-tmp) go vet ./internal/policies/... ./internal/fup/...
	@$(policies-with-disk-tmp) go vet -tags=integration ./internal/policies/... ./internal/fup/...
	@$(policies-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./internal/policies/... ./internal/fup/...
	@$(policies-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags=integration ./internal/policies/... ./internal/fup/...

policies-verify: policies-lint policies-build policies-test
