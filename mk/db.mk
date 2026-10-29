TEST_TARGETS += db-test
VERIFY_TARGETS += db-test

.PHONY: db-up db-down db-migrate db-test

db-up:
	docker compose -f deploy/dev/postgres/compose.yaml up -d --wait

db-down:
	docker compose -f deploy/dev/postgres/compose.yaml down

db-migrate:
	GOTOOLCHAIN=local CGO_ENABLED=0 go run ./internal/db/cmd/fvpn-migrate up

db-test:
	GOTOOLCHAIN=local CGO_ENABLED=0 go test -tags integration ./internal/db/... ./db/...
