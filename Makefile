include $(wildcard mk/*.mk)

.DEFAULT_GOAL := build
.PHONY: build test lint verify
build: $(BUILD_TARGETS)
test: $(TEST_TARGETS)
lint: $(LINT_TARGETS)
verify: $(VERIFY_TARGETS)
