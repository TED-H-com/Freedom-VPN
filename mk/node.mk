# Node agent area: fvpn-agent, fvpn-watchdog and their libraries.
#
# Works standalone (make -f mk/node.mk node-verify) and when the root Makefile
# includes mk/*.mk. Recipes run from the repository root.
#
#   node-build        linux/amd64 and linux/arm64 agent and watchdog
#   node-test         unit tests (no Docker, no netlab)
#   node-test-integration  agent proofs using FVPN_TEST_DATABASE_URL
#   node-lint         gofmt, vet and the same staticcheck pin as mk/go.mk
#   node-verify       lint, test and build
#   node-test-netlab  privileged lab; not part of node-test or TEST_TARGETS
#   node-test-live-netlab  foreground live-read latency/injection regression

NODE_GO   ?= go
# The shared host's /tmp is tmpfs. Go creates unique children and removes
# completed test/build scratch; keep their parent on the worktree filesystem.
NODE_TMPDIR := $(CURDIR)/.agent/tmp
NODE_ENV  := TMPDIR='$(NODE_TMPDIR)' GOTMPDIR='$(NODE_TMPDIR)' CGO_ENABLED=0 GOTOOLCHAIN=local
NODE_PKGS := ./node/agent/... ./node/watchdog/... ./node/internal/agentstore/... ./node/internal/controlclient/... ./node/internal/timeanchor/... ./node/internal/fakecontrol/... ./node/internal/nodelock/... ./node/internal/usagestore/... ./node/internal/meter/...
NODE_STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.8.2
NODE_LAB_PREFIX ?= fvpn-w2-agent-core-lab
NODE_NETLAB_TEST_ARGS ?=
NODE_NETLAB_MK := mk/netlab.mk

BUILD_TARGETS  += node-build
TEST_TARGETS   += node-test
LINT_TARGETS   += node-lint
VERIFY_TARGETS += node-verify

.PHONY: node-build node-test node-test-integration node-lint node-verify node-test-netlab
.PHONY: node-test-live-netlab node-temp

node-temp:
	@mkdir -p '$(NODE_TMPDIR)'

node-test-live-netlab:
	bash tests/netlab/agent/run-live.sh

node-build: node-temp
	$(NODE_ENV) GOOS=linux GOARCH=amd64 $(NODE_GO) build -trimpath -o bin/linux_amd64/fvpn-agent ./node/agent/cmd/fvpn-agent
	$(NODE_ENV) GOOS=linux GOARCH=arm64 $(NODE_GO) build -trimpath -o bin/linux_arm64/fvpn-agent ./node/agent/cmd/fvpn-agent
	$(NODE_ENV) GOOS=linux GOARCH=amd64 $(NODE_GO) build -trimpath -o bin/linux_amd64/fvpn-watchdog ./node/watchdog/cmd/fvpn-watchdog
	$(NODE_ENV) GOOS=linux GOARCH=arm64 $(NODE_GO) build -trimpath -o bin/linux_arm64/fvpn-watchdog ./node/watchdog/cmd/fvpn-watchdog

node-test: node-temp
	$(NODE_ENV) $(NODE_GO) test $(NODE_PKGS)

node-test-integration: node-temp
	$(NODE_ENV) $(NODE_GO) test -count=1 -tags integration ./node/agent/...

node-lint: node-temp
	@test -z "$$(gofmt -l node/agent node/watchdog node/internal/agentstore node/internal/controlclient node/internal/timeanchor node/internal/fakecontrol node/internal/nodelock node/internal/usagestore node/internal/meter tests/netlab/agent)" || { echo 'gofmt required'; exit 1; }
	$(NODE_ENV) $(NODE_GO) vet $(NODE_PKGS)
	$(NODE_ENV) $(NODE_GO) vet -tags netlab ./node/agent/... ./tests/netlab/agent/...
	$(NODE_ENV) $(NODE_GO) run $(NODE_STATICCHECK) $(NODE_PKGS)
	$(NODE_ENV) $(NODE_GO) run $(NODE_STATICCHECK) -tags netlab ./node/agent/... ./tests/netlab/agent/...

node-verify: node-lint node-test node-build

# Privileged. Brings the shared netlab container up under this task's name,
# runs only tests/netlab/agent, then removes the container and the lab image.
node-test-netlab: node-temp
	@set -eu; \
	status=0; \
	$(NODE_ENV) $(NODE_GO) build -trimpath -o bin/fvpn-agent ./node/agent/cmd/fvpn-agent || status=$$?; \
	if [ $$status -eq 0 ]; then $(NODE_ENV) $(NODE_GO) build -trimpath -o bin/fvpn-watchdog ./node/watchdog/cmd/fvpn-watchdog || status=$$?; fi; \
	if [ $$status -eq 0 ]; then $(NODE_ENV) $(NODE_GO) build -trimpath -o bin/fvpn-labcontrol ./node/internal/fakecontrol/cmd/fvpn-labcontrol || status=$$?; fi; \
	if [ $$status -eq 0 ]; then $(MAKE) -f $(NODE_NETLAB_MK) netlab-up FVPN_LAB_PREFIX='$(NODE_LAB_PREFIX)' || status=$$?; fi; \
	if [ $$status -eq 0 ]; then \
		docker cp bin/fvpn-agent $(NODE_LAB_PREFIX):/usr/local/bin/fvpn-agent || status=$$?; \
		docker cp bin/fvpn-watchdog $(NODE_LAB_PREFIX):/usr/local/bin/fvpn-watchdog || status=$$?; \
		docker cp bin/fvpn-labcontrol $(NODE_LAB_PREFIX):/usr/local/bin/fvpn-labcontrol || status=$$?; \
	fi; \
	if [ $$status -eq 0 ]; then \
		FVPN_NETLAB=1 FVPN_LAB_PREFIX='$(NODE_LAB_PREFIX)' \
			$(NODE_ENV) $(NODE_GO) test -tags netlab -count=1 -timeout 15m -p 1 -parallel 1 -v $(NODE_NETLAB_TEST_ARGS) ./tests/netlab/agent || status=$$?; \
	fi; \
	$(MAKE) -f $(NODE_NETLAB_MK) netlab-down FVPN_LAB_PREFIX='$(NODE_LAB_PREFIX)' || true; \
	docker rmi $(NODE_LAB_PREFIX):lab >/dev/null 2>&1 || true; \
	rm -f bin/fvpn-agent bin/fvpn-watchdog bin/fvpn-labcontrol; \
	exit $$status
