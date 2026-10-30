# Supporting component checks never imply §19 full-stack acceptance.
AT_SECOPS_RUN := python3 tests/e2e/at/secops/diskexec.py
export GOTOOLCHAIN := local
export CGO_ENABLED := 0
export PYTHONDONTWRITEBYTECODE := 1

LINT_TARGETS += at-secops-lint
TEST_TARGETS += at-secops-test

.PHONY: at-secops at-secops-test at-secops-components at-secops-lint at-secops-build

# Exits 2 for blocked acceptance; evidence is still written for all five ATs.
at-secops:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 python3 tests/e2e/at/secops/run.py

.PHONY: at-secops-followup3
at-secops-followup3:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 python3 tests/e2e/at/secops/run.py --follow-up-3

at-secops-components:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 python3 tests/e2e/at/secops/run.py --components-only

at-secops-test:
	$(AT_SECOPS_RUN) go test -tags e2e -p 1 -parallel 2 ./tests/e2e/at/secops/...
	$(AT_SECOPS_RUN) python3 -m unittest discover -s tests/e2e/at/secops -p 'test_*.py'

at-secops-lint:
	@test -z "$$(gofmt -l $$(git ls-files --cached --others --exclude-standard 'tests/e2e/at/secops/*.go'))" || { echo 'gofmt required'; exit 1; }
	$(AT_SECOPS_RUN) go vet -tags e2e,integration ./tests/e2e/at/secops/...
	$(AT_SECOPS_RUN) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e,integration ./tests/e2e/at/secops/...

at-secops-build:
	@mkdir -p .agent/secops/build
	GOOS=linux GOARCH=amd64 $(AT_SECOPS_RUN) go test -c -tags e2e,integration -o .agent/secops/build/amd64.test ./tests/e2e/at/secops
	GOOS=linux GOARCH=arm64 $(AT_SECOPS_RUN) go test -c -tags e2e,integration -o .agent/secops/build/arm64.test ./tests/e2e/at/secops

# Requires a fresh own-prefix e2e-up; leaves that stack running for inspection.
.PHONY: at-secops-pitr
at-secops-pitr:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 FVPN_SECOPS_PITR=1 $(AT_SECOPS_RUN) go test -v -tags e2e,integration -p 1 -parallel 2 -count=1 -timeout 12m -run '^TestAT30_LivePITRSource$$' ./tests/e2e/at/secops/...
	python3 tests/e2e/at/secops/pitr.py

.PHONY: at-secops-followup4
at-secops-followup4:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 python3 tests/e2e/at/secops/run.py --follow-up-4

.PHONY: at-secops-native-scale
at-secops-native-scale:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 python3 tests/e2e/at/secops/scale_run.py

.PHONY: at-secops-followup5
at-secops-followup5:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 python3 tests/e2e/at/secops/followup5_run.py

.PHONY: at-secops-followup6
at-secops-followup6:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 python3 tests/e2e/at/secops/followup6_run.py

.PHONY: at-secops-followup7
at-secops-followup7:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 python3 tests/e2e/at/secops/followup7_run.py

.PHONY: at-secops-followup8
at-secops-followup8:
	FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000 python3 tests/e2e/at/secops/followup8_run.py
