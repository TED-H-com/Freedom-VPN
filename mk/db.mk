# Standalone, private disk scratch; never use a tmpfs test basetemp.
db-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/db.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

TEST_TARGETS += db-test
VERIFY_TARGETS += db-test

.PHONY: db-up db-down db-migrate db-test

db-up:
	docker compose -f deploy/dev/postgres/compose.yaml up -d --wait

db-down:
	docker compose -f deploy/dev/postgres/compose.yaml down

db-migrate:
	@$(db-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run ./internal/db/cmd/fvpn-migrate up

db-test:
	@$(db-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test -tags integration ./internal/db/... ./db/...
