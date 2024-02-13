#
# Copyright 2023 Stacklok, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: clean-gen
clean-gen: ## clean generated files
	rm -rf $(shell find pkg/api -iname "*.go") & rm -rf $(shell find pkg/api -iname "*.swagger.json") & rm -rf pkg/api/protodocs

.PHONY: gen
gen: buf sqlc mock ## run code generation targets (buf, sqlc, mock)
	$(MAKE) authz-model

.PHONY: buf
buf: ## generate protobuf files
	buf generate

.PHONY: sqlc
sqlc: ## generate sqlc files
	sqlc generate

.PHONY: mock
mock: ## generate mocks
	mockgen -package mockdb -destination database/mock/store.go github.com/stacklok/minder/internal/db Store
	mockgen -package mockgh -destination internal/providers/github/mock/github.go -source pkg/providers/v1/providers.go GitHub
	mockgen -package auth -destination internal/auth/mock/jwtauth.go github.com/stacklok/minder/internal/auth JwtValidator,KeySetFetcher
	mockgen -package mockverify -destination internal/verifier/mock/verify.go github.com/stacklok/minder/internal/verifier/verifyif ArtifactVerifier

.PHONY: cli-docs
cli-docs: ## generate cli-docs
	@rm -rf docs/docs/ref/cli
	@mkdir -p docs/docs/ref/cli
	@echo 'label: Minder CLI' > docs/docs/ref/cli/_category_.yml
	@echo 'position: 20' >> docs/docs/ref/cli/_category_.yml
	@go run -tags '$(BUILDTAGS)' cmd/cli/main.go docs
