# mk/dns.mk: DNS policy engine, publisher, node applier, and dev resolvers

export GOTOOLCHAIN := local
export CGO_ENABLED := 0

FVPN_DNS_PREFIX ?= fvpn-dev-adguard
FVPN_HOST_UID ?= $(shell id -u)
FVPN_HOST_GID ?= $(shell id -g)
DNS_COMPOSE_DIR := deploy/dev/adguard
DNS_COMPOSE_FILE := $(DNS_COMPOSE_DIR)/docker-compose.yml
DNS_PKGS := ./internal/dnspolicy/... ./internal/dnspublish/... ./node/dns/...

dns-with-disk-tmp = set -eu; mkdir -p .agent/tmp; task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/go.XXXXXX"); trap 'rm -rf "$$task_tmp"' EXIT; export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp";

BUILD_TARGETS += dns-build
TEST_TARGETS += dns-test
LINT_TARGETS += dns-lint
VERIFY_TARGETS += dns-verify

.PHONY: dns-build dns-lint dns-verify dns-test dns-up dns-down dns-test-integration dns-test-publish dns-lab

dns-build:
	@$(dns-with-disk-tmp) go build $(DNS_PKGS) ./apps/worker/...
	@$(dns-with-disk-tmp) GOARCH=arm64 go build $(DNS_PKGS) ./apps/worker/...

dns-lint:
	@test -z "$$(gofmt -l internal/dnspolicy internal/dnspublish node/dns tests/netlab/dns apps/worker/jobs.go apps/worker/jobs_dns.go)" || (echo "gofmt needed on: $$(gofmt -l internal/dnspolicy internal/dnspublish node/dns tests/netlab/dns apps/worker/jobs.go apps/worker/jobs_dns.go)" && exit 1)
	@$(dns-with-disk-tmp) go vet $(DNS_PKGS)
	@$(dns-with-disk-tmp) go vet -tags=adguard_integration ./internal/dnspolicy/...
	@$(dns-with-disk-tmp) go vet -tags=integration ./internal/dnspublish/...
	@$(dns-with-disk-tmp) go vet -tags=netlab ./tests/netlab/dns/...
	@$(dns-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 $(DNS_PKGS)
	@$(dns-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags=adguard_integration ./internal/dnspolicy/...
	@$(dns-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags=integration ./internal/dnspublish/...
	@$(dns-with-disk-tmp) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags=netlab ./tests/netlab/dns/...

dns-verify: dns-lint dns-build dns-test

dns-test:
	CGO_ENABLED=0 GOTOOLCHAIN=local go test -count=1 $(DNS_PKGS)

dns-test-publish:
	CGO_ENABLED=0 GOTOOLCHAIN=local go test -count=1 -tags=integration ./internal/dnspublish/...

dns-lab:
	FVPN_NETLAB=1 FVPN_LAB_PREFIX=$${FVPN_LAB_PREFIX:-fvpn-w3-dns-publish} CGO_ENABLED=0 GOTOOLCHAIN=local go test -count=1 -tags=netlab -timeout 10m ./tests/netlab/dns/...

dns-up:
	FVPN_DNS_PREFIX=$(FVPN_DNS_PREFIX) bash $(DNS_COMPOSE_DIR)/bootstrap.sh
	FVPN_DNS_PREFIX=$(FVPN_DNS_PREFIX) FVPN_HOST_UID=$(FVPN_HOST_UID) FVPN_HOST_GID=$(FVPN_HOST_GID) docker compose -p $(FVPN_DNS_PREFIX) -f $(DNS_COMPOSE_FILE) up -d --wait

dns-down:
	-FVPN_DNS_PREFIX=$(FVPN_DNS_PREFIX) docker compose -p $(FVPN_DNS_PREFIX) -f $(DNS_COMPOSE_FILE) down -v --remove-orphans
	rm -rf $(DNS_COMPOSE_DIR)/.state

dns-test-integration:
	FVPN_DNS_PREFIX=$(FVPN_DNS_PREFIX) CGO_ENABLED=0 GOTOOLCHAIN=local go test -v -count=1 -tags=adguard_integration ./internal/dnspolicy/...
