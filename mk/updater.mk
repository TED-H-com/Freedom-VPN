# Updater area: node updater, update journal and release packaging.
#
# Works standalone (make -f mk/updater.mk updater-verify) and when the root
# Makefile includes mk/*.mk. Recipes run from the repository root.
#
#   updater-build   linux/amd64 and linux/arm64 binaries
#   updater-test    journal, updater and release tests
#   updater-lint    gofmt, vet and the same staticcheck pin as mk/go.mk
#   updater-verify  lint, test and build

UPDATER_GO   ?= go
UPDATER_ENV  := CGO_ENABLED=0 GOTOOLCHAIN=local
UPDATER_PKGS := ./node/updater/... ./node/internal/nodelock/... ./node/internal/updatejournal/... ./node/internal/healthattest/... ./packaging/release/...
UPDATER_STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.8.2

BUILD_TARGETS  += updater-build
TEST_TARGETS   += updater-test
LINT_TARGETS   += updater-lint
VERIFY_TARGETS += updater-verify

.PHONY: updater-build updater-test updater-lint updater-verify

updater-build:
	$(UPDATER_ENV) GOOS=linux GOARCH=amd64 $(UPDATER_GO) build -trimpath -o bin/linux_amd64/fvpn-updater ./node/updater/cmd/fvpn-updater
	$(UPDATER_ENV) GOOS=linux GOARCH=arm64 $(UPDATER_GO) build -trimpath -o bin/linux_arm64/fvpn-updater ./node/updater/cmd/fvpn-updater
	$(UPDATER_ENV) GOOS=linux GOARCH=amd64 $(UPDATER_GO) build -trimpath -o bin/linux_amd64/fvpn-release ./packaging/release/cmd/fvpn-release
	$(UPDATER_ENV) GOOS=linux GOARCH=arm64 $(UPDATER_GO) build -trimpath -o bin/linux_arm64/fvpn-release ./packaging/release/cmd/fvpn-release

updater-test:
	$(UPDATER_ENV) $(UPDATER_GO) test $(UPDATER_PKGS)

updater-lint:
	@test -z "$$(gofmt -l node/updater node/internal/nodelock node/internal/updatejournal node/internal/healthattest packaging/release)" || { echo 'gofmt required'; exit 1; }
	$(UPDATER_ENV) $(UPDATER_GO) vet $(UPDATER_PKGS)
	$(UPDATER_ENV) $(UPDATER_GO) run $(UPDATER_STATICCHECK) $(UPDATER_PKGS)

updater-verify: updater-lint updater-test updater-build
