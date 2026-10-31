# Contracts area: OpenAPI document, JSON Schemas, vectors, examples, generated
# Go/TypeScript types and the shared leaf packages (pkg/decstr, pkg/canonjson,
# pkg/envelope, pkg/apierr, pkg/api).
#
# Works standalone (make -f mk/contracts.mk contracts-verify) and when the
# root Makefile includes mk/*.mk. Recipes run from the repository root.
#
#   contracts-generate  regenerate schemas, vectors, signed examples, pkg/api and contracts/ts
#   contracts-verify    lint, build, drift, typecheck, test and coverage (everything below)
#   contracts-drift     regenerate into a temp dir and fail on any difference
#   contracts-lint      Redocly lint (recommended-strict, zero problems)
#   contracts-test      Go tests for the contract packages and the TypeScript vector check
#   contracts-cover     coverage of the four leaf packages (fails below 85%)
#   contracts-build     vet and cross-compile for linux/amd64 and linux/arm64

CONTRACTS_OAPI_CODEGEN    ?= github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.1
CONTRACTS_OPENAPI_TS      ?= openapi-typescript@7.13.0
CONTRACTS_REDOCLY         ?= @redocly/cli@2.72.1
CONTRACTS_TYPESCRIPT      ?= typescript@5.9.3
CONTRACTS_STATICCHECK     ?= honnef.co/go/tools/cmd/staticcheck@v0.8.2
CONTRACTS_GO              ?= go
CONTRACTS_NPX             ?= npx --yes
CONTRACTS_NODE            ?= node
CONTRACTS_GO_PKGS         := ./pkg/decstr/... ./pkg/canonjson/... ./pkg/envelope/... ./pkg/apierr/... ./pkg/api/... ./contracts/...
CONTRACTS_LEAF_PKGS       := ./pkg/decstr ./pkg/canonjson ./pkg/envelope ./pkg/apierr
CONTRACTS_MIN_COVERAGE    ?= 85
CONTRACTS_ENV             := bash contracts/codegen/with-disk-tmp.sh env CGO_ENABLED=0 GOTOOLCHAIN=local

BUILD_TARGETS  += contracts-build
TEST_TARGETS   += contracts-test
LINT_TARGETS   += contracts-lint
VERIFY_TARGETS += contracts-verify

.PHONY: contracts-generate contracts-verify contracts-drift contracts-lint contracts-test contracts-cover contracts-build contracts-typecheck

contracts-generate:
	$(CONTRACTS_ENV) $(CONTRACTS_GO) run ./contracts/internal/cmd/contractsgen
	$(CONTRACTS_ENV) $(CONTRACTS_GO) run $(CONTRACTS_OAPI_CODEGEN) -config contracts/codegen/oapi-codegen.yaml contracts/openapi.yaml
	$(CONTRACTS_NPX) $(CONTRACTS_OPENAPI_TS) contracts/openapi.yaml -o contracts/ts/openapi.d.ts

contracts-verify:
	@$(MAKE) --no-print-directory -f mk/contracts.mk contracts-lint contracts-build contracts-drift contracts-typecheck contracts-test contracts-cover

contracts-drift:
	@$(CONTRACTS_ENV) bash -ec 'tmp="$$TMPDIR"; \
	echo "contracts: checking generated schemas, vectors and examples"; \
	$(CONTRACTS_ENV) $(CONTRACTS_GO) run ./contracts/internal/cmd/contractsgen -check; \
	echo "contracts: checking pkg/api drift"; \
	sed "s|^output: .*|output: $$tmp/api.gen.go|" contracts/codegen/oapi-codegen.yaml > "$$tmp/oapi-codegen.yaml"; \
	$(CONTRACTS_ENV) $(CONTRACTS_GO) run $(CONTRACTS_OAPI_CODEGEN) -config "$$tmp/oapi-codegen.yaml" contracts/openapi.yaml; \
	diff -u pkg/api/api.gen.go "$$tmp/api.gen.go"; \
	echo "contracts: checking contracts/ts drift"; \
	$(CONTRACTS_NPX) $(CONTRACTS_OPENAPI_TS) contracts/openapi.yaml -o "$$tmp/openapi.d.ts" >/dev/null; \
	diff -u contracts/ts/openapi.d.ts "$$tmp/openapi.d.ts"; \
	echo "contracts: generated files are up to date"'

contracts-lint:
	$(CONTRACTS_NPX) $(CONTRACTS_REDOCLY) lint contracts/openapi.yaml --config contracts/redocly.yaml --max-problems 1000
	@test -z "$$(gofmt -l pkg/decstr pkg/canonjson pkg/envelope pkg/apierr pkg/api contracts)" || { gofmt -l pkg/decstr pkg/canonjson pkg/envelope pkg/apierr pkg/api contracts; echo "contracts: gofmt needed"; exit 1; }

	$(CONTRACTS_ENV) $(CONTRACTS_GO) vet $(CONTRACTS_GO_PKGS)
	$(CONTRACTS_ENV) $(CONTRACTS_GO) run $(CONTRACTS_STATICCHECK) $(CONTRACTS_GO_PKGS)

contracts-typecheck:
	$(CONTRACTS_NPX) -p $(CONTRACTS_TYPESCRIPT) tsc --noEmit --strict contracts/ts/openapi.d.ts contracts/ts/schema.d.ts contracts/ts/revision13.typecheck.ts contracts/ts/revision14.typecheck.ts
	$(CONTRACTS_NPX) -p $(CONTRACTS_TYPESCRIPT) tsc --noEmit --strict --target es2022 contracts/ts/strict-json.ts

contracts-test:
	$(CONTRACTS_ENV) $(CONTRACTS_GO) test -count=1 $(CONTRACTS_GO_PKGS)
	$(CONTRACTS_NODE) contracts/ts/verify-vectors.ts

contracts-cover:
	@set -e; for p in $(CONTRACTS_LEAF_PKGS); do \
	  out=$$($(CONTRACTS_ENV) $(CONTRACTS_GO) test -count=1 -cover $$p); echo "$$out"; \
	  pct=$$(echo "$$out" | sed -n 's/.*coverage: \([0-9.]*\)%.*/\1/p'); \
	  awk -v p="$$pct" -v m=$(CONTRACTS_MIN_COVERAGE) -v n="$$p" 'BEGIN { if (p == "" || p+0 < m+0) { printf "contracts: %s coverage %s%% is below %s%%\n", n, p, m; exit 1 } }'; \
	done

contracts-build:
	$(CONTRACTS_ENV) $(CONTRACTS_GO) vet $(CONTRACTS_GO_PKGS)
	$(CONTRACTS_ENV) GOOS=linux GOARCH=amd64 $(CONTRACTS_GO) build $(CONTRACTS_GO_PKGS)
	$(CONTRACTS_ENV) GOOS=linux GOARCH=arm64 $(CONTRACTS_GO) build $(CONTRACTS_GO_PKGS)
