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

projectname?=minder

# Profile and Rule examples
include examples/Makefile

# Unfortunately, we need OS detection for docker-compose
OS := $(shell uname -s)

COMPOSE?=docker-compose
CONTAINER?=docker

# Services to run in docker-compose. Defaults to all
services?=

# Additional arguments to pass to docker-compose
COMPOSE_ARGS?=-d

# Additional flags and env vars for ko
KO_DOCKER_REPO?=ko.local
KO_PUSH_IMAGE?=false
KO_PLATFORMS=linux/amd64,linux/arm64
HELM_PACKAGE_VERSION?=0.1.0

TARGET_ENV?=staging
BUILDTAGS?=$(TARGET_ENV)

default: help

.PHONY: help
help: ## list makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: gen
gen: buf sqlc mock ## Run code generation targets (buf, sqlc, mock)

.PHONY: buf
buf:  ## Run protobuf code generation
	buf generate

.PHONY: clean-gen
clean-gen:
	rm -rf $(shell find pkg/api -iname "*.go") & rm -rf $(shell find pkg/api -iname "*.swagger.json") & rm -rf pkg/api/protodocs

.PHONY: cli-docs
cli-docs:
	@rm -rf docs/docs/ref/cli/commands
	@mkdir -p docs/docs/ref/cli/commands
	@echo 'label: Commands' > docs/docs/ref/cli/commands/_category_.yml
	@echo 'position: 20' >> docs/docs/ref/cli/commands/_category_.yml
	@go run -tags '$(BUILDTAGS)' cmd/cli/main.go docs

.PHONY: build
build: ## build golang binary
	CGO_ENABLED=0 go build \
		-trimpath \
		-tags '$(BUILDTAGS)' \
		-ldflags "-X github.com/stacklok/minder/internal/constants.CLIVersion=$(shell git describe --abbrev=0 --tags)+ref.$(shell git rev-parse --short HEAD)" \
		-o ./bin/minder ./cmd/cli
	CGO_ENABLED=0 go build -trimpath -tags '$(BUILDTAGS)' -o ./bin/$(projectname)-server ./cmd/server
	CGO_ENABLED=0 go build -trimpath -tags '$(BUILDTAGS)' -o ./bin/medev ./cmd/dev

.PHONY: run-cli
run-cli: ## run the CLI, needs additional arguments
	@go run -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)" -tags '$(BUILDTAGS)' ./cmd/cli

.PHONY: run-server
run-server: ## run the app
	@go run -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)" -tags '$(BUILDTAGS)' ./cmd/server serve

DOCKERARCH := $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

.PHONY: run-docker
run-docker:  ## run the app under docker.
	# podman (at least) doesn't seem to like multi-arch images, and sometimes picks the wrong one (e.g. amd64 on arm64)
	# We also need to remove the build: directives to use ko builds
	# ko resolve will fill in the image: field in the compose file, but it adds a yaml document separator
	sed -e '/^  *build:/d'  -e 's|  image: minder:latest|  image: ko://github.com/stacklok/minder/cmd/server|' docker-compose.yaml | ko resolve --base-import-paths --platform linux/$(DOCKERARCH) -f - | sed 's/^--*$$//' > .resolved-compose.yaml
	@echo "Running docker-compose up $(services)"
	$(COMPOSE) -f .resolved-compose.yaml down && $(COMPOSE) -f .resolved-compose.yaml up $(COMPOSE_ARGS) $(services)
	rm .resolved-compose.yaml*

.PHONY: helm
helm:  ## build the helm chart to a local archive, using ko for the image build
	cd deployment/helm; \
	    ko resolve --platform=${KO_PLATFORMS} --base-import-paths --push=${KO_PUSH_IMAGE} -f values.yaml > values.tmp.yaml && \
		mv values.tmp.yaml values.yaml && \
		helm dependency update && \
		helm package --version="${HELM_PACKAGE_VERSION}" . && \
		cat values.yaml
	git checkout deployment/helm/values.yaml

.PHONY: helm-template
helm-template: ## renders the helm templates which is useful for debugging
	cd deployment/helm; \
		helm dependency update && \
		helm template .

.PHONY: bootstrap
bootstrap: ## install build deps
	go generate -tags tools tools/tools.go
	# N.B. each line runs in a different subshell, so we don't need to undo the 'cd' here
	cd tools && go mod tidy && go install \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
			google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc \
			github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc \
			github.com/sqlc-dev/sqlc
	# Create a config.yaml if it doesn't exist
	# TODO: remove this when all config is handled in internal/config
	cp -n config/config.yaml.example ./config.yaml || echo "config.yaml already exists, not overwriting"
	# Create keys:
	mkdir -p .ssh
	@echo "Generating token key passphrase"
	openssl rand -base64 32 > .ssh/token_key_passphrase
	# Make sure the keys are readable by the docker user
	chmod 644 .ssh/*

.PHONY: test
test: clean init-examples ## display test coverage
	go test -json -race -v ./... | gotestfmt

.PHONY: clean
clean: ## clean up environment
	rm -rf dist/* & rm -rf bin/*

.PHONY: cover
cover: ## display test coverage
	go test -v -race $(shell go list ./... | grep -v /vendor/) -v -coverprofile=coverage.out
	go tool cover -func=coverage.out

.PHONY: lint
lint: ## lint go files
	golangci-lint run

.PHONY: pre-commit
pre-commit:	## run pre-commit hooks
	pre-commit run --all-files

.PHONY: sqlc
sqlc: ## generate sqlc files
	sqlc generate

.PHONY: migrateup
migrateup: ## run migrate up
	@go run -tags '$(BUILDTAGS)' cmd/server/main.go migrate up --yes

.PHONY: migratedown
migratedown: ## run migrate down
	@go run -tags '$(BUILDTAGS)' cmd/server/main.go migrate down

.PHONY: dbschema
dbschema:	## generate database schema with schema spy, monitor file until doc is created and copy it
	mkdir -p database/schema/output && chmod a+w database/schema/output
	cd database/schema && $(COMPOSE) run -u 1001:1001 --rm schemaspy -configFile /config/schemaspy.properties -imageformat png
	sleep 10
	cp database/schema/output/diagrams/summary/relationships.real.large.png docs/static/img/minder/schema.png
	cd database/schema && $(COMPOSE) down -v && rm -rf output

.PHONY: mock
mock:  ## generate mocks
	mockgen -package mockdb -destination database/mock/store.go github.com/stacklok/minder/internal/db Store
	mockgen -package mockgh -destination internal/providers/github/mock/github.go -source pkg/providers/v1/providers.go GitHub
	mockgen -package auth -destination internal/auth/mock/jwtauth.go github.com/stacklok/minder/internal/auth JwtValidator,KeySetFetcher

.PHONY: github-login
github-login:  ## setup GitHub login on Keycloak
ifndef KC_GITHUB_CLIENT_ID
	$(error KC_GITHUB_CLIENT_ID is not set)
endif
ifndef KC_GITHUB_CLIENT_SECRET
	$(error KC_GITHUB_CLIENT_SECRET is not set)
endif
	$(CONTAINER) exec -it keycloak_container /opt/keycloak/bin/kcadm.sh create identity-provider/instances -r stacklok -s alias=github -s providerId=github -s enabled=true  -s 'config.useJwksUrl="true"' -s config.clientId=$$KC_GITHUB_CLIENT_ID -s config.clientSecret=$$KC_GITHUB_CLIENT_SECRET
