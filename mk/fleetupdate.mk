export GOTOOLCHAIN := local
export CGO_ENABLED := 0

fleetupdate-with-disk-tmp = set -eu; mkdir -p .agent/tmp; export TMPDIR="$$(pwd)/.agent/tmp" GOTMPDIR="$$(pwd)/.agent/tmp";

BUILD_TARGETS += fleetupdate-build
TEST_TARGETS += fleetupdate-test
LINT_TARGETS += fleetupdate-lint
VERIFY_TARGETS += fleetupdate-verify

.PHONY: fleetupdate-build fleetupdate-test fleetupdate-integration fleetupdate-lint fleetupdate-verify
fleetupdate-build:
	@$(fleetupdate-with-disk-tmp) for arch in amd64 arm64; do GOOS=linux GOARCH=$$arch go build ./internal/releases/... ./internal/fleetupdate/... ./internal/installerjob/...; done
fleetupdate-test:
	@$(fleetupdate-with-disk-tmp) go test ./internal/releases/... ./internal/fleetupdate/... ./internal/installerjob/...
fleetupdate-integration:
	@$(fleetupdate-with-disk-tmp) go test -count=1 -tags integration -p 1 -parallel 2 ./internal/fleetupdate/... ./internal/installerjob/... ./internal/releases/... ./internal/db/...
fleetupdate-lint:
	@test -z "$$(gofmt -l $$(rg --files internal/releases internal/fleetupdate internal/installerjob -g '*.go'))"
	@$(fleetupdate-with-disk-tmp) go vet ./internal/releases/... ./internal/fleetupdate/... ./internal/installerjob/...
	@$(fleetupdate-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./internal/releases/... ./internal/fleetupdate/... ./internal/installerjob/...
fleetupdate-verify: fleetupdate-lint fleetupdate-test fleetupdate-build fleetupdate-integration
