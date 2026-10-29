# Standalone, private disk scratch; never use a tmpfs test basetemp.
nodes-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/nodes.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

NODES_PKGS := ./internal/nodes/... ./internal/desiredstate/... ./internal/exits/... ./internal/leases/...
NODES_GO_FILES := internal/nodes internal/desiredstate internal/exits internal/leases

BUILD_TARGETS += nodes-build
TEST_TARGETS += nodes-test
LINT_TARGETS += nodes-lint
VERIFY_TARGETS += nodes-verify

.PHONY: nodes-build nodes-test nodes-lint nodes-verify

nodes-build:
	@$(nodes-with-disk-tmp) for arch in amd64 arm64; do \
		GOOS=linux GOARCH=$$arch GOTOOLCHAIN=local CGO_ENABLED=0 go build $(NODES_PKGS); \
	done

nodes-test:
	@$(nodes-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test -tags integration $(NODES_PKGS)

nodes-lint:
	@test -z "$$(gofmt -l $(NODES_GO_FILES))" || { echo 'gofmt required in nodes packages'; exit 1; }
	@$(nodes-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go vet $(NODES_PKGS)
	@$(nodes-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 $(NODES_PKGS)

nodes-verify: nodes-lint nodes-test nodes-build
