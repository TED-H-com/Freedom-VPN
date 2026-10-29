# Works standalone and when included by the repository Makefile.
NETBIRD_TMPDIR ?= $(CURDIR)/.agent/netbird-tmp
NETBIRD_COMPOSE = docker compose -f deploy/dev/netbird/compose.yml
export FVPN_NB_PREFIX ?= fvpn-dev-netbird
export GOTOOLCHAIN = local
export CGO_ENABLED = 0
TEST_TARGETS += netbird-test
LINT_TARGETS += netbird-vet
.PHONY: netbird-up netbird-bootstrap netbird-down netbird-reset netbird-test netbird-vet netbird-test-integration
netbird-up:
	python3 -B deploy/dev/netbird/prepare.py
	$(NETBIRD_COMPOSE) --profile clients build
	$(NETBIRD_COMPOSE) up -d server dashboard gateway
netbird-bootstrap:
	python3 -B deploy/dev/netbird/bootstrap.py
	$(NETBIRD_COMPOSE) --profile clients up -d exit member-a member-b replay
	python3 -B deploy/dev/netbird/login-members.py
netbird-down:
	$(NETBIRD_COMPOSE) --profile clients down
netbird-reset:
	python3 -B deploy/dev/netbird/reset.py
netbird-test:
	mkdir -p "$(NETBIRD_TMPDIR)"
	TMPDIR="$(NETBIRD_TMPDIR)" go test ./internal/netbird/...
netbird-vet:
	mkdir -p "$(NETBIRD_TMPDIR)"
	TMPDIR="$(NETBIRD_TMPDIR)" go vet ./internal/netbird/...
	TMPDIR="$(NETBIRD_TMPDIR)" go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./internal/netbird/...
netbird-test-integration:
	mkdir -p "$(NETBIRD_TMPDIR)"
	TMPDIR="$(NETBIRD_TMPDIR)" FVPN_NETBIRD_API_URL="$${FVPN_NETBIRD_API_URL:-http://127.0.0.1:$${FVPN_NB_API_PORT:-18080}/api}" \
	FVPN_NETBIRD_TOKEN_FILE="$${FVPN_NETBIRD_TOKEN_FILE:-$(CURDIR)/deploy/dev/netbird/.state/adapter-token}" \
	FVPN_NETBIRD_CLIENT_TESTS=1 go test -count=1 -tags netbird_integration -v ./internal/netbird/...
