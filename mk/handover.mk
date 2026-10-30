# Standalone, private disk scratch; never use a tmpfs test basetemp.
handover-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/handover.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

BUILD_TARGETS += handover-build
TEST_TARGETS += handover-test
LINT_TARGETS += handover-lint
VERIFY_TARGETS += handover-verify

.PHONY: handover-build handover-test handover-test-integration handover-lint handover-verify

handover-build:
	@$(handover-with-disk-tmp) for arch in amd64 arm64; do \
		GOOS=linux GOARCH=$$arch GOTOOLCHAIN=local CGO_ENABLED=0 go test -c -o /dev/null ./tests/handover/...; \
		GOOS=linux GOARCH=$$arch GOTOOLCHAIN=local CGO_ENABLED=0 go test -tags integration -c -o /dev/null ./tests/handover/...; \
	done

handover-test:
	@$(handover-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test -count=1 ./tests/handover/...

handover-test-integration:
	@$(handover-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test -count=1 -tags integration ./tests/handover/...

handover-lint:
	@test -z "$$(gofmt -l tests/handover)" || { echo 'gofmt required in tests/handover'; exit 1; }
	@$(handover-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go vet ./tests/handover/...
	@$(handover-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go vet -tags=integration ./tests/handover/...
	@$(handover-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/handover/...
	@$(handover-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags=integration ./tests/handover/...
	@$(handover-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test -count=1 ./tests/handover/...

handover-verify: handover-lint handover-build handover-test handover-test-integration
