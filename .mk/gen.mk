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
	mockgen -package mockghhook -destination internal/repositories/github/webhooks/mock/manager.go github.com/stacklok/minder/internal/repositories/github/webhooks WebhookManager
	mockgen -package mockcrypto -destination internal/crypto/mock/crypto.go github.com/stacklok/minder/internal/crypto Engine
	mockgen -package mockevents -destination internal/events/mock/eventer.go github.com/stacklok/minder/internal/events Interface
	mockgen -package mockghclients -destination internal/repositories/github/clients/mock/clients.go github.com/stacklok/minder/internal/repositories/github/clients GitHubRepoClient
	mockgen -package mockghrepo -destination internal/repositories/github/mock/service.go github.com/stacklok/minder/internal/repositories/github RepositoryService
	mockgen -package mockbundle -destination internal/marketplaces/bundles/mock/reader.go github.com/stacklok/minder/pkg/mindpak/reader BundleReader
	mockgen -package mockprofsvc -destination internal/profiles/mock/service.go github.com/stacklok/minder/internal/profiles ProfileService
	mockgen -package mockrulesvc -destination internal/ruletypes/mock/service.go github.com/stacklok/minder/internal/ruletypes RuleTypeService
	mockgen -package mockbundle -destination internal/marketplaces/bundles/mock/source.go github.com/stacklok/minder/pkg/mindpak/sources BundleSource
	mockgen -package mocksubscription -destination internal/marketplaces/subscriptions/mock/subscription.go github.com/stacklok/minder/internal/marketplaces/subscriptions SubscriptionService
	mockgen -package mockprofsvc -destination internal/providers/mock/service.go github.com/stacklok/minder/internal/providers ProviderService
	mockgen -package mockratecache -destination internal/providers/ratecache/mock/restcache.go github.com/stacklok/minder/internal/providers/ratecache RestClientCache
	mockgen -package mockgh -destination internal/providers/github/mock/common.go github.com/stacklok/minder/internal/providers/github ClientService


# Ugly hack: cobra uses tabs for code blocks in markdown in some places
# This leads to some issues with MDX in the docs renderer
# Use sed to get rid of lines which begin with tabs and swap in backticks.
.PHONY: cli-docs
cli-docs: ## generate cli-docs
	$(eval DOC_PATH := docs/docs/ref/cli)
	@rm -rf ${DOC_PATH}
	@mkdir -p docs/docs/ref/cli
	@echo 'label: CLI Commands' > ${DOC_PATH}/_category_.yml
	@echo 'position: 20' >> ${DOC_PATH}/_category_.yml
	@go run -tags '$(BUILDTAGS)' cmd/cli/main.go docs
	@# this sed is much uglier than it should be so that it can run on Mac
	@sed -i.bak 's/^	\(.*\)$$/```\$\n\1\$\n```/g' ${DOC_PATH}/minder_completion_zsh.md ${DOC_PATH}/minder_completion_bash.md
	@# clean up temporary files created by sed
	@rm ${DOC_PATH}/*.bak
