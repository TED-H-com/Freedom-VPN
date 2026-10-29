export GOTOOLCHAIN := local
export CGO_ENABLED := 0

pki-packages := ./internal/pki/... ./internal/enrollment/... ./internal/signing/...
pki-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/pki.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

BUILD_TARGETS += pki-build
TEST_TARGETS += pki-test
LINT_TARGETS += pki-lint
VERIFY_TARGETS += pki-verify

.PHONY: pki-build pki-test pki-test-integration pki-lint pki-verify
pki-build:
	@$(pki-with-disk-tmp) for arch in amd64 arm64; do GOOS=linux GOARCH=$$arch go build $(pki-packages); done
pki-test:
	@$(pki-with-disk-tmp) go test $(pki-packages)
pki-test-integration:
	@$(pki-with-disk-tmp) go test -count=1 -tags integration $(pki-packages)
pki-lint:
	@test -z "$$(gofmt -l internal/pki internal/enrollment internal/signing)"
	@$(pki-with-disk-tmp) go vet $(pki-packages)
	@$(pki-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 $(pki-packages)
pki-verify: pki-lint pki-test pki-build
