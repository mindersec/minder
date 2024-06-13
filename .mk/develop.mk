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

DOCKERARCH := $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

RUN_DOCKER_NO_TEARDOWN?=false

YQ_BUILD_REPLACE_STRING := 'del(.services.minder.build) | \
.services.minder.image |= "ko://github.com/stacklok/minder/cmd/server" | \
del(.services.migrate.build) | \
.services.migrate.image |= "ko://github.com/stacklok/minder/cmd/server"'

.PHONY: run-cli
run-cli: ## run the CLI, needs additional arguments
	@go run -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)" -tags '$(BUILDTAGS)' ./cmd/cli

.PHONY: run-server
run-server: ## run the app
	@go run -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)" -tags '$(BUILDTAGS)' ./cmd/server serve

.PHONY: run-docker-teardown
run-docker-teardown: ## teardown the docker compose environment
ifeq ($(RUN_DOCKER_NO_TEARDOWN),false)
	@echo "Running docker compose down"
	@$(COMPOSE) down
else
	@echo "Skipping docker compose down"
endif

.PHONY: run-docker
run-docker: run-docker-teardown ## run the app under docker compose
	@echo "Running docker compose up $(services)..."
	@echo "Building the minder-server image (KO_DOCKER_REPO=$(KO_DOCKER_REPO))..."

	@# podman (at least) doesn't seem to like multi-arch images, and sometimes picks the wrong one (e.g. amd64 on arm64)
	@# We also need to remove the build: directives to use ko builds
	@# ko resolve will fill in the image: field in the compose file, but it adds a yaml document separator
	@yq e $(YQ_BUILD_REPLACE_STRING) docker-compose.yaml | KO_DOCKER_REPO=$(KO_DOCKER_REPO) ko resolve --base-import-paths --platform linux/$(DOCKERARCH) -f - | sed 's/^--*$$//' > .resolved-compose.yaml
	@$(COMPOSE) -f .resolved-compose.yaml up $(COMPOSE_ARGS) $(services)
	@rm .resolved-compose.yaml*

.PHONY: stop-docker
stop-docker: ## stop the app under docker compose
	@echo "Running docker compose down $(services)..."
	@$(COMPOSE) down

.PHONY: pre-commit
pre-commit:	## run pre-commit hooks
	pre-commit run --all-files

.PHONY: bootstrap
bootstrap: ## install build deps
	cd tools && go generate -tags tools ./tools.go
	# N.B. each line runs in a different subshell, so we don't need to undo the 'cd' here
	cd tools && go mod tidy && go install \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
			google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc \
			github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc \
			github.com/sqlc-dev/sqlc \
			github.com/norwoodj/helm-docs/cmd/helm-docs \
			github.com/openfga/cli/cmd/fga \
			go.uber.org/mock/mockgen \
			github.com/mikefarah/yq/v4
	# Create a config.yaml and server-config.yaml if they don't exist
	# TODO: remove this when all config is handled in internal/config
	cp -n config/config.yaml.example ./config.yaml || echo "config.yaml already exists, not overwriting"
	cp -n config/server-config.yaml.example ./server-config.yaml || echo "server-config.yaml already exists, not overwriting"
	# Create keys:
	mkdir -p .ssh
	@echo "Generating token key passphrase"
	openssl rand -base64 32 > .ssh/token_key_passphrase
	# Make sure the keys are readable by the docker user
	chmod 644 .ssh/*

.PHONY: generate-encryption-key
generate-encryption-key:	## Generates an encryption key that's useful for encrypting minder secrets.
	@openssl rand -base64 32

.PHONY: lint-fix
lint-fix: ## fix all linting issues which can be automatically fixed
	golangci-lint run --fix

