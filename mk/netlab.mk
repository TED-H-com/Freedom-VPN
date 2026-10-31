# Network lab targets. Names start with netlab- so a root Makefile can include
# this file next to other areas. netlab-test is privileged and is not added to
# TEST_TARGETS. netlab-unit is the unprivileged library test.
#
#   make -f mk/netlab.mk netlab-test
#   FVPN_LAB_PREFIX=fvpn-w1-netlab make -f mk/netlab.mk netlab-test

FVPN_LAB_PREFIX ?= fvpn-netlab
NETLAB_MK := $(lastword $(MAKEFILE_LIST))
ROOT := $(abspath $(dir $(NETLAB_MK))/..)
NETLAB_IMAGE := $(FVPN_LAB_PREFIX):lab
NETLAB_SCRATCH := $(ROOT)/.agent/tmp/$(FVPN_LAB_PREFIX)-scratch
NETLAB_BIN := $(ROOT)/.agent/tmp/$(FVPN_LAB_PREFIX)-fvpn-netlab
# User-defined bridge. Docker installs embedded DNS NAT for 127.0.0.11 in
# the container netns. topology.sh drops the default route that comes with it.
NETLAB_DNS_NET := $(FVPN_LAB_PREFIX)-dns

.PHONY: netlab-unit netlab-lint netlab-build netlab-image netlab-wipe-scratch netlab-up netlab-down netlab-test netlab-test-full netlab-evidence

# gofmt, vet, and staticcheck for the enforce library and the netlab harness.
# The privileged suite is not part of this target.
netlab-lint:
	@set -eu; \
	cd $(ROOT); \
	files=$$(git ls-files --cached --others --exclude-standard 'node/enforce/*.go' 'tests/netlab/*.go' 'tests/netlab/cmd/fvpn-netlab/*.go' 'tests/netlab/at/*.go' 'tests/netlab/agent/*.go'); \
	test -z "$$(gofmt -l $$files)" || { echo 'gofmt required'; gofmt -l $$files; exit 1; }; \
	mkdir -p .agent/tmp; \
	task_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/go.XXXXXX"); \
	trap 'rm -rf "$$task_tmp"' EXIT; \
	export TMPDIR="$$task_tmp" GOTMPDIR="$$task_tmp" CGO_ENABLED=0 GOTOOLCHAIN=local; \
	go vet ./node/enforce/...; \
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./node/enforce/...; \
	go vet -tags netlab ./tests/netlab ./tests/netlab/at ./tests/netlab/agent ./tests/netlab/cmd/fvpn-netlab; \
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -tags netlab ./tests/netlab ./tests/netlab/at ./tests/netlab/agent ./tests/netlab/cmd/fvpn-netlab

netlab-unit:
	cd $(ROOT) && CGO_ENABLED=0 go test ./node/enforce/...

netlab-build:
	cd $(ROOT) && CGO_ENABLED=0 go build -o $(NETLAB_BIN) ./tests/netlab/cmd/fvpn-netlab

netlab-image:
	docker build -t $(NETLAB_IMAGE) $(ROOT)/tests/netlab

# Keys under the scratch mount are created as root inside the container.
# The host user cannot unlink them. Wipe through a root container.
netlab-wipe-scratch:
	mkdir -p $(NETLAB_SCRATCH)
	docker rm -f $(FVPN_LAB_PREFIX)-wipe >/dev/null 2>&1 || true
	docker run --rm --name $(FVPN_LAB_PREFIX)-wipe --network none \
		-v $(NETLAB_SCRATCH):/scratch $(NETLAB_IMAGE) \
		find /scratch -mindepth 1 -delete

netlab-up: netlab-build netlab-image
	docker rm -f $(FVPN_LAB_PREFIX) >/dev/null 2>&1 || true
	$(MAKE) -f $(NETLAB_MK) netlab-wipe-scratch FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)'
	docker network inspect $(NETLAB_DNS_NET) >/dev/null 2>&1 || docker network create $(NETLAB_DNS_NET)
	docker run -d --name $(FVPN_LAB_PREFIX) --privileged --network $(NETLAB_DNS_NET) \
		--cap-add NET_ADMIN --cap-add SYS_TIME \
		-v $(ROOT):/work:ro \
		-v $(NETLAB_SCRATCH):/scratch \
		$(NETLAB_IMAGE)
	docker cp $(NETLAB_BIN) $(FVPN_LAB_PREFIX):/usr/local/bin/fvpn-netlab
	docker exec $(FVPN_LAB_PREFIX) bash /work/tests/netlab/scripts/topology.sh

netlab-down:
	docker rm -f $(FVPN_LAB_PREFIX) >/dev/null 2>&1 || true
	docker network rm $(NETLAB_DNS_NET) >/dev/null 2>&1 || true
	$(MAKE) -f $(NETLAB_MK) netlab-wipe-scratch FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)'
	rmdir $(NETLAB_SCRATCH) 2>/dev/null || true

netlab-test:
	@set -eu; \
	status=0; \
	$(MAKE) -f $(NETLAB_MK) netlab-up FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' || status=$$?; \
	if [ $$status -eq 0 ]; then \
		cd $(ROOT) && FVPN_NETLAB=1 FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' \
			go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 -v ./tests/netlab || status=$$?; \
	fi; \
	$(MAKE) -f $(NETLAB_MK) netlab-down FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' || true; \
	exit $$status

# Refresh the tracked Gate G2 evidence. Other netlab targets leave evidence
# in the test's temporary directory.
netlab-evidence:
	@set -eu; \
	status=0; \
	$(MAKE) -f $(NETLAB_MK) netlab-up FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' || status=$$?; \
	if [ $$status -eq 0 ]; then \
		cd $(ROOT) && FVPN_NETLAB=1 FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' \
			FVPN_NETLAB_EVIDENCE_DIR='docs/poc/G2-evidence' \
			go test -tags netlab -count=1 -timeout 40m -p 1 -parallel 1 -v ./tests/netlab || status=$$?; \
	fi; \
	$(MAKE) -f $(NETLAB_MK) netlab-down FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' || true; \
	exit $$status

netlab-test-full:
	@set -eu; \
	status=0; \
	$(MAKE) -f $(NETLAB_MK) netlab-up FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' || status=$$?; \
	if [ $$status -eq 0 ]; then \
		cd $(ROOT) && FVPN_NETLAB=1 FVPN_NETLAB_FULL=1 FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' \
			go test -tags netlab -count=1 -timeout 100m -p 1 -parallel 1 -v ./tests/netlab || status=$$?; \
	fi; \
	$(MAKE) -f $(NETLAB_MK) netlab-down FVPN_LAB_PREFIX='$(FVPN_LAB_PREFIX)' || true; \
	exit $$status

TEST_TARGETS += netlab-unit
