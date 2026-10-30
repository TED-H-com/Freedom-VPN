# Standalone backup makefile fragment (`mk/backup.mk`).
# Provides backup-lint, backup-test, backup-drill, and backup-drill-clean (§15.4, §17.2, SECOPS-05, AT30).
backup-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/backup.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

LINT_TARGETS += backup-lint
TEST_TARGETS += backup-test
VERIFY_TARGETS += backup-lint backup-test

.PHONY: backup-lint backup-test backup-drill backup-drill-clean

# Runs bash -n over all backup shell scripts (plus shellcheck/shfmt if installed),
# and runs gofmt, go vet, and pinned staticcheck v0.8.2 over tests/backup.
backup-lint:
	@for f in deploy/backup/*.sh deploy/backup/postgres/*.sh deploy/backup/test/*.sh; do \
		if [ -f "$$f" ]; then bash -n "$$f"; fi; \
	done
	@if command -v shellcheck >/dev/null 2>&1; then shellcheck deploy/backup/*.sh deploy/backup/postgres/*.sh deploy/backup/test/*.sh; fi
	@if command -v shfmt >/dev/null 2>&1; then shfmt -d deploy/backup/*.sh deploy/backup/postgres/*.sh deploy/backup/test/*.sh; fi
	@test -z "$$(gofmt -l $$(git ls-files --cached --others --exclude-standard 'tests/backup/*.go' 'tests/backup/**/*.go'))" || { echo 'gofmt required'; exit 1; }
	@$(backup-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go vet ./tests/backup/...
	@$(backup-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go vet -tags integration ./tests/backup/...
	@$(backup-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/backup/...
	@$(backup-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags integration ./tests/backup/...

backup-test:
	@$(backup-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test -v -count=1 ./tests/backup/...

backup-drill:
	@$(backup-with-disk-tmp) GOTOOLCHAIN=local CGO_ENABLED=0 go test -v -tags integration -count=1 -timeout 10m -run '^TestAT30_BackupPITRDrill$$' ./tests/backup/...

# Removes stale drill objects named fvpn-w4-ops-backups or fvpn-w4-ops-backups-*.
# Not used by backup-drill. The test deletes only its own FVPN_BACKUP_DRILL_PREFIX.
# Do not run this while a drill is in progress.
backup-drill-clean:
	@set -eu; \
	family=fvpn-w4-ops-backups; \
	mkdir -p .agent/tmp; \
	list=$$(mktemp "$$(pwd)/.agent/tmp/backup-drill-clean.XXXXXX"); \
	trap 'rm -f "$$list"' EXIT; \
	echo "removing stale Docker objects named $$family-*"; \
	clean_matching() { \
		kind=$$1; shift; \
		while IFS= read -r name || [ -n "$$name" ]; do \
			name=$${name#/}; \
			case "$$name" in \
			"" ) ;; \
			$$family|$$family-*) \
				echo "$$kind $$name"; \
				"$$@" "$$name" || true ;; \
			esac; \
		done < "$$list"; \
	}; \
	docker ps -a --format '{{.Names}}' > "$$list"; \
	clean_matching container docker rm -fv; \
	docker volume ls --format '{{.Name}}' > "$$list"; \
	clean_matching volume docker volume rm -f; \
	docker network ls --format '{{.Name}}' > "$$list"; \
	clean_matching network docker network rm; \
	docker images --format '{{.Repository}}:{{.Tag}}' > "$$list"; \
	clean_matching image docker rmi -f

.PHONY: backup-review-proof
backup-review-proof:
	@$(backup-with-disk-tmp) bash deploy/backup/test/review_proof.sh
