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
NETLAB_SCRATCH := /tmp/$(FVPN_LAB_PREFIX)-scratch
NETLAB_BIN := /tmp/$(FVPN_LAB_PREFIX)-fvpn-netlab

.PHONY: netlab-unit netlab-build netlab-image netlab-up netlab-down netlab-test netlab-test-full

netlab-unit:
	cd $(ROOT) && CGO_ENABLED=0 go test ./node/enforce/...

netlab-build:
	cd $(ROOT) && CGO_ENABLED=0 go build -o $(NETLAB_BIN) ./tests/netlab/cmd/fvpn-netlab

netlab-image:
	docker build -t $(NETLAB_IMAGE) $(ROOT)/tests/netlab

netlab-up: netlab-build netlab-image
	docker rm -f $(FVPN_LAB_PREFIX) >/dev/null 2>&1 || true
	rm -rf $(NETLAB_SCRATCH)
	mkdir -p $(NETLAB_SCRATCH)
	docker run -d --name $(FVPN_LAB_PREFIX) --privileged --network none \
		--cap-add NET_ADMIN --cap-add SYS_TIME \
		-v $(ROOT):/work:ro \
		-v $(NETLAB_SCRATCH):/scratch \
		$(NETLAB_IMAGE)
	docker cp $(NETLAB_BIN) $(FVPN_LAB_PREFIX):/usr/local/bin/fvpn-netlab
	docker exec $(FVPN_LAB_PREFIX) bash /work/tests/netlab/scripts/topology.sh

netlab-down:
	docker rm -f $(FVPN_LAB_PREFIX) >/dev/null 2>&1 || true
	rm -rf $(NETLAB_SCRATCH)

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
