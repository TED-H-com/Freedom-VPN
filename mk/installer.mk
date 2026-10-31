# Ubuntu node installer: fvpn-node-install.
#
# Works standalone (make -f mk/installer.mk installer-verify) and when the
# root Makefile includes mk/*.mk. Recipes run from the repository root.
#
#   installer-build   linux/amd64 and linux/arm64 fvpn-node-install
#   installer-unit    library tests (no Docker)
#   installer-lint    gofmt, vet and staticcheck, including -tags installer
#   installer-verify  lint, unit and build
#   installer-test    privileged systemd containers; not part of TEST_TARGETS
#                     or of the GitHub-hosted workflow

INSTALLER_GO  ?= go
INSTALLER_ENV := CGO_ENABLED=0 GOTOOLCHAIN=local
INSTALLER_STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.8.2
INSTALLER_PREFIX ?= fvpn-w5-node-installer-

BUILD_TARGETS  += installer-build
TEST_TARGETS   += installer-unit
LINT_TARGETS   += installer-lint
VERIFY_TARGETS += installer-verify

.PHONY: installer-build installer-unit installer-lint installer-verify installer-test

installer-build:
	$(INSTALLER_ENV) GOOS=linux GOARCH=amd64 $(INSTALLER_GO) build -trimpath -o bin/linux_amd64/fvpn-node-install ./packaging/installer/cmd/fvpn-node-install
	$(INSTALLER_ENV) GOOS=linux GOARCH=arm64 $(INSTALLER_GO) build -trimpath -o bin/linux_arm64/fvpn-node-install ./packaging/installer/cmd/fvpn-node-install

installer-unit:
	$(INSTALLER_ENV) $(INSTALLER_GO) test ./packaging/installer/

installer-lint:
	@test -z "$$(gofmt -l packaging/installer deploy/node tests/installer)" || { echo 'gofmt required'; exit 1; }
	$(INSTALLER_ENV) $(INSTALLER_GO) vet ./packaging/installer/...
	$(INSTALLER_ENV) $(INSTALLER_GO) vet -tags installer ./tests/installer/...
	$(INSTALLER_ENV) $(INSTALLER_GO) run $(INSTALLER_STATICCHECK) ./packaging/installer/...
	$(INSTALLER_ENV) $(INSTALLER_GO) run $(INSTALLER_STATICCHECK) -tags installer ./tests/installer/...
	$(INSTALLER_ENV) INSTALLER_GO=$(INSTALLER_GO) python3 tests/installer/control-build.py lint

installer-verify: installer-lint installer-unit installer-build

# Privileged systemd-as-PID-1 suite. Local only. Names stay under
# $(INSTALLER_PREFIX). Not run on GitHub-hosted runners.
installer-test:
	$(INSTALLER_ENV) $(INSTALLER_GO) test -v -tags installer -count=1 -timeout 40m -p 1 -parallel 1 ./tests/installer/
