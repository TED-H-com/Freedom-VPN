export GOTOOLCHAIN := local
export CGO_ENABLED := 0

BUILD_TARGETS += go-build
TEST_TARGETS += go-test
LINT_TARGETS += go-lint
VERIFY_TARGETS += go-verify

.PHONY: go-build go-test go-test-integration go-lint go-verify go-tidy-check

go-build:
	@set -eu; for arch in amd64 arm64; do \
		packages="$$(GOOS=linux GOARCH=$$arch go list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./...)"; \
		mkdir -p bin/linux_$$arch; \
		for package in $$packages; do \
			name=$${package##*/}; \
			echo "building $$package for linux/$$arch"; \
			GOOS=linux GOARCH=$$arch go build -trimpath -o bin/linux_$$arch/$$name $$package; \
		done; \
	done

go-test:
	go test ./...

go-test-integration:
	go test -tags integration ./...

go-lint:
	@test -z "$$(gofmt -l $$(git ls-files --cached --others --exclude-standard '*.go'))" || { echo 'gofmt required'; exit 1; }
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...

go-tidy-check:
	go mod tidy -diff

go-verify: go-lint go-test go-build go-tidy-check
