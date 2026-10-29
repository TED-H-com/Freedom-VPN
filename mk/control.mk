# Standalone and root aggregate targets for the control framework.
export GOTOOLCHAIN := local
export CGO_ENABLED := 0
CONTROL_PACKAGES := ./apps/control/... ./apps/worker/... ./internal/httpapi/... ./internal/auth/... ./internal/operations/... ./internal/audit/... ./internal/config/...
CONTROL_CODEGEN := github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1
CONTROL_STATICCHECK := honnef.co/go/tools/cmd/staticcheck@v0.8.2
control-with-disk-tmp = set -eu; mkdir -p .agent/tmp; control_tmp=$$(mktemp -d "$$(pwd)/.agent/tmp/control.XXXXXX"); trap 'rm -rf "$$control_tmp"' EXIT; export TMPDIR="$$control_tmp" GOTMPDIR="$$control_tmp";
BUILD_TARGETS += control-build
TEST_TARGETS += control-test
LINT_TARGETS += control-lint
VERIFY_TARGETS += control-verify
.PHONY: control-generate control-drift control-test control-test-integration control-build control-lint control-verify
control-generate:
	@$(control-with-disk-tmp) go run $(CONTROL_CODEGEN) -config internal/httpapi/gen/oapi-codegen.yaml contracts/openapi.yaml; go run ./internal/httpapi/cmd/controlgen
control-drift:
	@$(control-with-disk-tmp) mkdir -p "$$control_tmp/gen"; \
	sed "s|^output: .*|output: $$control_tmp/gen/server.gen.go|" internal/httpapi/gen/oapi-codegen.yaml > "$$control_tmp/oapi.yaml"; \
	go run $(CONTROL_CODEGEN) -config "$$control_tmp/oapi.yaml" contracts/openapi.yaml; \
	go run ./internal/httpapi/cmd/controlgen -out "$$control_tmp"; \
	diff -u internal/httpapi/gen/server.gen.go "$$control_tmp/gen/server.gen.go"; \
	diff -u internal/httpapi/gen/models.gen.go "$$control_tmp/gen/models.gen.go"; \
	diff -u internal/httpapi/composition.gen.go "$$control_tmp/composition.gen.go"; \
	echo 'control: generated files are current'
control-test:
	@$(control-with-disk-tmp) go test $(CONTROL_PACKAGES)
control-test-integration:
	@$(control-with-disk-tmp) go test -tags integration $(CONTROL_PACKAGES)
control-build:
	@$(control-with-disk-tmp) for arch in amd64 arm64; do GOOS=linux GOARCH=$$arch go build $(CONTROL_PACKAGES); echo "control: linux/$$arch build passed"; done
control-lint:
	@test -z "$$(gofmt -l apps/control apps/worker internal/httpapi internal/auth internal/operations internal/audit internal/config)"
	@$(control-with-disk-tmp) go vet $(CONTROL_PACKAGES); go run $(CONTROL_STATICCHECK) $(CONTROL_PACKAGES)
control-verify:
	@$(MAKE) --no-print-directory -f mk/control.mk control-drift control-lint control-test control-build
