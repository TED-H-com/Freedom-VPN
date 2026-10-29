# Standalone, private disk scratch; never use a tmpfs test basetemp.
db-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/db.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

LINT_TARGETS += db-lint
TEST_TARGETS += db-test
VERIFY_TARGETS += db-test

.PHONY: db-up db-down db-migrate db-test db-lint

db-up:
	docker compose -f deploy/dev/postgres/compose.yaml up -d --wait

db-down:
	docker compose -f deploy/dev/postgres/compose.yaml down

db-migrate:
	@$(db-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run ./internal/db/cmd/fvpn-migrate up

db-test:
	@$(db-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test -tags integration ./internal/db/... ./db/...

# Match the whole-module go-lint gate, including its pinned staticcheck.
db-lint:
	@test -z "$$(gofmt -l $$(git ls-files --cached --others --exclude-standard 'internal/db/*.go' 'internal/db/**/*.go' 'db/**/*.go'))" || { echo 'gofmt required'; exit 1; }
	@$(db-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go vet ./internal/db/... ./db/...
	@$(db-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./internal/db/... ./db/...
