# mk/dns.mk: DNS domain policy engine, AdGuard Home backend and dev resolvers

export GOTOOLCHAIN := local
export CGO_ENABLED := 0

FVPN_DNS_PREFIX ?= fvpn-dev-adguard
FVPN_HOST_UID ?= $(shell id -u)
FVPN_HOST_GID ?= $(shell id -g)
DNS_COMPOSE_DIR := deploy/dev/adguard
DNS_COMPOSE_FILE := $(DNS_COMPOSE_DIR)/docker-compose.yml

dns-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/go.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

BUILD_TARGETS += dns-build
TEST_TARGETS += dns-test
LINT_TARGETS += dns-lint
VERIFY_TARGETS += dns-verify

.PHONY: dns-build dns-lint dns-verify dns-test dns-up dns-down dns-test-integration

dns-build:
	@$(dns-with-disk-tmp) go build ./internal/dnspolicy/...
	@$(dns-with-disk-tmp) GOARCH=arm64 go build ./internal/dnspolicy/...

dns-lint:
	@test -z "$$(gofmt -l internal/dnspolicy)" || (echo "gofmt needed on: $$(gofmt -l internal/dnspolicy)" && exit 1)
	@$(dns-with-disk-tmp) go vet ./internal/dnspolicy/...
	@$(dns-with-disk-tmp) go vet -tags=adguard_integration ./internal/dnspolicy/...
	@$(dns-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./internal/dnspolicy/...

dns-verify: dns-lint dns-build dns-test

dns-test:
	CGO_ENABLED=0 GOTOOLCHAIN=local go test -count=1 ./internal/dnspolicy/...

dns-up:
	FVPN_DNS_PREFIX=$(FVPN_DNS_PREFIX) bash $(DNS_COMPOSE_DIR)/bootstrap.sh
	FVPN_DNS_PREFIX=$(FVPN_DNS_PREFIX) FVPN_HOST_UID=$(FVPN_HOST_UID) FVPN_HOST_GID=$(FVPN_HOST_GID) docker compose -p $(FVPN_DNS_PREFIX) -f $(DNS_COMPOSE_FILE) up -d --wait

dns-down:
	-FVPN_DNS_PREFIX=$(FVPN_DNS_PREFIX) docker compose -p $(FVPN_DNS_PREFIX) -f $(DNS_COMPOSE_FILE) down -v --remove-orphans
	rm -rf $(DNS_COMPOSE_DIR)/.state

dns-test-integration:
	FVPN_DNS_PREFIX=$(FVPN_DNS_PREFIX) CGO_ENABLED=0 GOTOOLCHAIN=local go test -v -count=1 -tags=adguard_integration ./internal/dnspolicy/...
