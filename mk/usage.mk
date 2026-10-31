# Standalone, private disk scratch; never use a tmpfs test basetemp.
usage-with-disk-tmp = set -eu; mkdir -p .agent/tmp; export TMPDIR="$$(pwd)/.agent/tmp" GOTMPDIR="$$(pwd)/.agent/tmp";

BUILD_TARGETS += usage-build
TEST_TARGETS += usage-test
LINT_TARGETS += usage-lint
VERIFY_TARGETS += usage-verify

.PHONY: usage-build usage-test usage-test-integration usage-lint usage-verify

usage-build:
	@$(usage-with-disk-tmp) for arch in amd64 arm64; do \
		GOOS=linux GOARCH=$$arch GOTOOLCHAIN=local CGO_ENABLED=0 go build ./internal/usage/... ./internal/audit/...; \
	done

usage-test:
	@$(usage-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test ./internal/usage/... ./internal/audit/...

usage-test-integration:
	@$(usage-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test -tags integration ./internal/usage/... ./internal/audit/...

usage-lint:
	@test -z "$$(gofmt -l internal/usage internal/audit)" || { echo 'gofmt required in internal/usage or internal/audit'; exit 1; }
	@$(usage-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go vet ./internal/usage/... ./internal/audit/...
	@$(usage-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go vet -tags=integration ./internal/usage/... ./internal/audit/...
	@$(usage-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./internal/usage/... ./internal/audit/...
	@$(usage-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags=integration ./internal/usage/... ./internal/audit/...

usage-verify: usage-lint usage-build usage-test usage-test-integration
