# Standalone member-domain lint; keep the staticcheck pin aligned with mk/go.mk.
export GOTOOLCHAIN := local
export CGO_ENABLED := 0
members-packages := ./internal/members/... ./internal/entitlement/... ./internal/devices/... ./internal/netbird/... ./internal/config/... ./apps/control/... ./apps/worker/...
members-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/members.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";
LINT_TARGETS += members-lint
.PHONY: members-lint
members-lint:
	@test -z "$$(gofmt -l internal/members internal/entitlement internal/devices internal/netbird apps/control/wire_members.go apps/worker/jobs_members.go internal/config/members*.go)" || { echo 'gofmt required'; exit 1; }
	@$(members-with-disk-tmp) go vet $(members-packages)
	@$(members-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 $(members-packages)
