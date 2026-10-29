PORTAL_DIR := apps/portal

BUILD_TARGETS += portal-build
TEST_TARGETS += portal-test
LINT_TARGETS += portal-lint
VERIFY_TARGETS += portal-verify

.PHONY: portal-install portal-lint portal-test portal-build portal-dev-stack portal-e2e portal-verify

portal-install:
	cd $(PORTAL_DIR) && npm ci

portal-lint:
	cd $(PORTAL_DIR) && npm run typecheck
	cd $(PORTAL_DIR) && npm run lint
	cd $(PORTAL_DIR) && npm run format:check

portal-test:
	cd $(PORTAL_DIR) && npm run test:coverage

portal-build:
	cd $(PORTAL_DIR) && npm run build

portal-dev-stack:
	cd $(PORTAL_DIR) && npm run dev:stack

portal-e2e: portal-build
	cd $(PORTAL_DIR) && npm run e2e; status=$$?; rm -rf test-results playwright-report; exit $$status

portal-verify: portal-install portal-lint portal-test portal-build
