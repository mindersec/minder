FGA_TESTS_DIR := internal/authz/model/tests
FGA_TEST_FILES := $(wildcard $(FGA_TESTS_DIR)/*.tests.yaml)

.PHONY: authz-tests
authz-tests: ## run authz tests
	@for file in $(FGA_TEST_FILES); do \
		echo "* Running tests in $$file"; \
		fga model test --tests $$file; \
		echo ""; \
	done
