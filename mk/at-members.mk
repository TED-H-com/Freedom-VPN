export GOTOOLCHAIN := local
export CGO_ENABLED := 0

# Assigned live acceptance; incomplete ATs are non-green.
# Not added to offline TEST_TARGETS or VERIFY_TARGETS.
at-members-disk = set -eu; mkdir -p .agent/tmp; at_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/at-members.XXXXXX"); trap 'rm -rf "$$at_tmp"' EXIT; export TMPDIR="$$at_tmp" GOTMPDIR="$$at_tmp";

LINT_TARGETS += at-members-lint

.PHONY: at-members at-members-test at-members-lint at-members-build

at-members:
	@$(at-members-disk) FVPN_E2E=1 FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 go test -tags e2e -count=1 -p 1 -parallel 2 -timeout 30m -v ./tests/e2e/at/members/...

at-members-test:
	@$(at-members-disk) go test -p 1 -parallel 2 ./tests/e2e/at/members/...

at-members-lint:
	@test -z "$$(gofmt -l $$(git ls-files --cached --others --exclude-standard 'tests/e2e/at/members/*.go'))" || { echo 'gofmt required'; exit 1; }
	@$(at-members-disk) go vet -tags e2e ./tests/e2e/at/members/...
	@$(at-members-disk) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/at/members/...

at-members-build:
	@$(at-members-disk) for arch in amd64 arm64; do GOOS=linux GOARCH=$$arch go build -o "$$at_tmp/probe-$$arch" ./tests/e2e/at/members/cmd/probe; done
