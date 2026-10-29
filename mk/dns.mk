# mk/dns.mk: DNS domain policy engine, AdGuard Home backend and dev resolvers

FVPN_DNS_PREFIX ?= fvpn-dev-adguard
FVPN_HOST_UID ?= $(shell id -u)
FVPN_HOST_GID ?= $(shell id -g)
DNS_COMPOSE_DIR := deploy/dev/adguard
DNS_COMPOSE_FILE := $(DNS_COMPOSE_DIR)/docker-compose.yml

BUILD_TARGETS += dns-build
TEST_TARGETS += dns-test
LINT_TARGETS += dns-lint
VERIFY_TARGETS += dns-verify

.PHONY: dns-build dns-lint dns-verify dns-test dns-up dns-down dns-test-integration

dns-build:
	CGO_ENABLED=0 GOTOOLCHAIN=local go build ./internal/dnspolicy/...
	CGO_ENABLED=0 GOTOOLCHAIN=local GOARCH=arm64 go build ./internal/dnspolicy/...

dns-lint:
	@test -z "$$(gofmt -l internal/dnspolicy)" || (echo "gofmt needed on: $$(gofmt -l internal/dnspolicy)" && exit 1)
	CGO_ENABLED=0 GOTOOLCHAIN=local go vet ./internal/dnspolicy/...
	CGO_ENABLED=0 GOTOOLCHAIN=local go vet -tags=adguard_integration ./internal/dnspolicy/...

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
