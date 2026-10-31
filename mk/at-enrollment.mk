# AT06/AT07 acceptance. Live Docker work is manual-only and records non-green ATs.
export GOTOOLCHAIN := local
export CGO_ENABLED := 0

LINT_TARGETS += at-enrollment-lint

at-enrollment-disk = python3 tests/e2e/at/enrollment/disktmp.py

.PHONY: at-enrollment at-enrollment-lint

at-enrollment:
	@PYTHONDONTWRITEBYTECODE=1 python3 tests/e2e/at/enrollment/run.py

at-enrollment-lint:
	@test -z "$$(gofmt -l tests/e2e/at/enrollment)" || { echo 'gofmt required'; exit 1; }
	@$(at-enrollment-disk) sh -eu -c ' \
	go vet ./tests/e2e/at/enrollment; \
	go vet -tags e2e ./tests/e2e/at/enrollment; \
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./tests/e2e/at/enrollment; \
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/at/enrollment; \
	FVPN_E2E=0 go test -count=1 -tags e2e ./tests/e2e/at/enrollment; \
	python3 -m unittest discover -s tests/e2e/at/enrollment -p 'test_*.py'; \
	for arch in amd64 arm64; do GOOS=linux GOARCH=$$arch go test -tags e2e -c -o "$$TMPDIR/enrollment-$$arch.test" ./tests/e2e/at/enrollment; done; \
	echo "at-enrollment-lint: gofmt, vet, pinned staticcheck, offline checks and amd64/arm64 compilation passed"'
