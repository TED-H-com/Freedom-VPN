export GOTOOLCHAIN := local
export CGO_ENABLED := 0

fleetupdate-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/fleetupdate.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

BUILD_TARGETS += fleetupdate-build
TEST_TARGETS += fleetupdate-test
LINT_TARGETS += fleetupdate-lint
VERIFY_TARGETS += fleetupdate-verify

.PHONY: fleetupdate-build fleetupdate-test fleetupdate-integration fleetupdate-lint fleetupdate-verify
fleetupdate-build:
	@$(fleetupdate-with-disk-tmp) for arch in amd64 arm64; do GOOS=linux GOARCH=$$arch go build ./internal/releases/... ./internal/fleetupdate/...; done
fleetupdate-test:
	@$(fleetupdate-with-disk-tmp) go test ./internal/releases/... ./internal/fleetupdate/...
fleetupdate-integration:
	@$(fleetupdate-with-disk-tmp) go test -count=1 -tags integration -p 1 -parallel 2 ./internal/fleetupdate/... ./internal/releases/... ./internal/db/...
fleetupdate-lint:
	@test -z "$$(gofmt -l $$(find internal/releases internal/fleetupdate -name '*.go'))"
	@$(fleetupdate-with-disk-tmp) go vet ./internal/releases/... ./internal/fleetupdate/...
	@$(fleetupdate-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./internal/releases/... ./internal/fleetupdate/...
fleetupdate-verify: fleetupdate-lint fleetupdate-test fleetupdate-build fleetupdate-integration
