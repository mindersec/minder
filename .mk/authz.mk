FGA_TESTS_DIR := internal/authz/model/tests
FGA_TEST_FILES := $(wildcard $(FGA_TESTS_DIR)/*.tests.yaml)

FGA_MODEL := internal/authz/model/minder.fga
FGA_JSON_MODEL := internal/authz/model/minder.generated.json

.PHONY: authz-tests
authz-tests: authz-validate ## run authz tests
	@for file in $(FGA_TEST_FILES); do \
		echo "* Running tests in $$file"; \
		fga model test --tests $$file; \
		echo ""; \
	done

.PHONY: authz-validate
authz-validate:  ## validate the authz model
	@echo "* Validating authz model"
	@fga model validate --file internal/authz/model/minder.fga

.PHONY: authz-model
authz-model: authz-validate $(FGA_JSON_MODEL) ## validate and generate the authz model

$(FGA_JSON_MODEL): $(FGA_MODEL)
	@echo "* Generating JSON model"
	@fga model transform --file $(FGA_MODEL) | tee $(FGA_JSON_MODEL)