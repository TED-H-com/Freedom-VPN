export GOTOOLCHAIN := local
export CGO_ENABLED := 0

# Assigned live acceptance; incomplete ATs are non-green.
# Not added to offline TEST_TARGETS or VERIFY_TARGETS.
at-members-disk = python3 tests/e2e/at/members/disktmp.py

LINT_TARGETS += at-members-lint

.PHONY: at-members at-members-test at-members-lint at-members-build at-members-readback

at-members:
	@python3 tests/e2e/at/members/admission.py
	@$(at-members-disk) env FVPN_E2E=1 FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 go test -tags e2e -count=1 -p 1 -parallel 2 -timeout 55m -v ./tests/e2e/at/members/...

at-members-test:
	@$(at-members-disk) go test -p 1 -parallel 2 ./tests/e2e/at/members/...

at-members-lint:
	@test -z "$$(gofmt -l $$(git ls-files --cached --others --exclude-standard 'tests/e2e/at/members/*.go'))" || { echo 'gofmt required'; exit 1; }
	@$(at-members-disk) go vet -tags e2e ./tests/e2e/at/members/...
	@$(at-members-disk) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags e2e ./tests/e2e/at/members/...

at-members-build:
	@$(at-members-disk) sh -ec 'for arch in amd64 arm64; do for pkg in ./tests/e2e/at/members/cmd/probe ./tests/e2e/cmd/fvpn-e2e-fixture-probe ./tests/e2e/cmd/fvpn-e2e-member-session; do GOOS=linux GOARCH=$$arch go build -o "$$AT_MEMBERS_TMP/$${pkg##*/}-$$arch" $$pkg; done; done'

# Supplements evidence from the same live run; leaves complete AT status blocked.
at-members-readback:
	@$(at-members-disk) env FVPN_E2E=1 FVPN_AT_MEMBERS_READBACK=1 FVPN_E2E_PREFIX=w4-members FVPN_E2E_PORT_BASE=24000 go test -tags e2e -run '^TestATMembers_FollowupReadback$$' -count=1 -p 1 -parallel 2 -timeout 15m -v ./tests/e2e/at/members
