# Usage agent area: node meter, usage store and usage upload.
#
# Works standalone (make -f mk/usage-agent.mk usage-agent-verify) and when the
# root Makefile includes mk/*.mk. Recipes run from the repository root.
#
#   usage-agent-build   linux/amd64 and linux/arm64 compile of the packages
#   usage-agent-test    store, meter, upload and the usage snapshot command
#   usage-agent-lint    gofmt, vet and the same staticcheck pin as mk/go.mk
#   usage-agent-verify  lint, test and build

USAGE_AGENT_GO ?= go
USAGE_AGENT_ENV := CGO_ENABLED=0 GOTOOLCHAIN=local
USAGE_AGENT_PKGS := ./node/internal/meter/... ./node/internal/usagestore/... ./node/internal/usageupload/... ./node/agent/cmd/fvpn-agent/...
USAGE_AGENT_STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.8.2

BUILD_TARGETS += usage-agent-build
TEST_TARGETS += usage-agent-test
LINT_TARGETS += usage-agent-lint
VERIFY_TARGETS += usage-agent-verify

.PHONY: usage-agent-build usage-agent-test usage-agent-lint usage-agent-verify

usage-agent-build:
	$(USAGE_AGENT_ENV) GOOS=linux GOARCH=amd64 $(USAGE_AGENT_GO) build $(USAGE_AGENT_PKGS)
	$(USAGE_AGENT_ENV) GOOS=linux GOARCH=arm64 $(USAGE_AGENT_GO) build $(USAGE_AGENT_PKGS)

usage-agent-test:
	$(USAGE_AGENT_ENV) $(USAGE_AGENT_GO) test $(USAGE_AGENT_PKGS)

usage-agent-lint:
	@test -z "$$(gofmt -l node/internal/meter node/internal/usagestore node/internal/usageupload node/agent/usage_snapshot.go node/agent/cmd/fvpn-agent)" || { echo 'gofmt required'; exit 1; }
	$(USAGE_AGENT_ENV) $(USAGE_AGENT_GO) vet $(USAGE_AGENT_PKGS)
	$(USAGE_AGENT_ENV) $(USAGE_AGENT_GO) run $(USAGE_AGENT_STATICCHECK) $(USAGE_AGENT_PKGS)

usage-agent-verify: usage-agent-lint usage-agent-test usage-agent-build
