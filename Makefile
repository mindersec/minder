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

projectname?=mediator

# Unfortunately, we need OS detection for docker-compose
OS := $(shell uname -s)

# Container runtime detection
ifeq (, $(shell which podman-compose))
    COMPOSE?=docker-compose
else
    COMPOSE?=podman-compose
endif

# Services to run in docker-compose. Defaults to all
services?=

# Additional arguments to pass to docker-compose
COMPOSE_ARGS?=-d

# Additional flags and env vars for ko
KO_DOCKER_REPO?=ko.local
KO_PUSH_IMAGE?=false
KO_PLATFORMS=linux/amd64,linux/arm64

default: help

.PHONY: help gen clean-gen build run-cli run-server bootstrap test clean cover lint pre-commit migrateup migratedown sqlc mock cli-docs

help: ## list makefile targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

gen: ## generate protobuf files
	buf generate

clean-gen:
	rm -rf $(shell find pkg/generated -iname "*.go") & rm -rf $(shell find pkg/generated -iname "*.swagger.json") & rm -rf pkg/generated/protodocs

cli-docs:
	@mkdir -p docs/docs/cli
	@go run cmd/cli/main.go docs

build: ## build golang binary
	# @go build -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)" -o bin/$(projectname)
	CGO_ENABLED=0 go build -trimpath -o ./bin/medic ./cmd/cli
	CGO_ENABLED=0 go build -trimpath -o ./bin/$(projectname)-server ./cmd/server

run-cli: ## run the CLI, needs additional arguments
	@go run -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)"  ./cmd/cli

run-server: ## run the app
	@go run -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags)"  ./cmd/server serve

DOCKERARCH := $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

run-docker:  ## run the app under docker.
	# podman (at least) doesn't seem to like multi-arch images, and sometimes picks the wrong one (e.g. amd64 on arm64)
	# We also need to remove the build: directives to use ko builds
	# ko resolve will fill in the image: field in the compose file, but it adds a yaml document separator
	sed -e '/^  *build:/d'  -e 's|  image: mediator:latest|  image: ko://github.com/stacklok/mediator/cmd/server|' docker-compose.yaml | ko resolve --base-import-paths --platform linux/$(DOCKERARCH) -f - | sed 's/^--*$$//' > .resolved-compose.yaml
	# MacOS can't tolerate the ":z" flag, Linux needs it.  https://github.com/containers/podman-compose/issues/509
ifeq ($(OS),Darwin)
	sed -i '' 's/:z$$//' .resolved-compose.yaml
endif
	@echo "Running docker-compose up $(services)"
	$(COMPOSE) -f .resolved-compose.yaml down && $(COMPOSE) -f .resolved-compose.yaml up $(COMPOSE_ARGS) $(services)
	rm .resolved-compose.yaml*

helm:  ## build the helm chart to a local archive, using ko for the image build
	cd deployment/helm; rm -f templates/combined.yml && \
	    ko resolve --platform=${KO_PLATFORMS} --base-import-paths --push=${KO_PUSH_IMAGE} -f templates/ > templates/combined.yml && \
		helm dependency update && \
		helm package .

bootstrap: ## install build deps
	go generate -tags tools tools/tools.go
	# N.B. each line runs in a different subshell, so we don't need to undo the 'cd' here
	cd tools && go mod tidy && go install \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
		github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
			google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/grpc/cmd/protoc-gen-go-grpc \
			github.com/kyleconroy/sqlc
	# Create a config.yaml if it doesn't exist
	cp -n config/config.yaml.example ./config.yaml || echo "config.yaml already exists, not overwriting"
	# Create keys:
	mkdir -p .ssh
	# No passphrase (-N), don't overwrite existing keys ("n" to prompt)
	echo n | ssh-keygen -t rsa -b 2048 -N "" -m PEM -f .ssh/access_token_rsa || true
	echo n | ssh-keygen -t rsa -b 2048 -N "" -m PEM -f .ssh/refresh_token_rsa || true
	openssl rsa -in .ssh/access_token_rsa -pubout -outform PEM -out .ssh/access_token_rsa.pub
	openssl rsa -in .ssh/refresh_token_rsa -pubout -outform PEM -out .ssh/refresh_token_rsa.pub
	# Make sure the keys are readable by the docker user
	chmod 644 .ssh/*

test: clean ## display test coverage
	go test -json -v ./... | gotestfmt

clean: ## clean up environment
	rm -rf dist/* & rm -rf bin/*

cover: ## display test coverage
	go test -v -race $(shell go list ./... | grep -v /vendor/) -v -coverprofile=coverage.out
	go tool cover -func=coverage.out

lint: ## lint go files
	golangci-lint run

pre-commit:	## run pre-commit hooks
	pre-commit run --all-files

sqlc: ## generate sqlc files
	sqlc generate

migrateup: ## run migrate up
	@go run cmd/server/main.go migrate up --yes     
migratedown: ## run migrate down
	@go run cmd/server/main.go migrate down

dbschema:	## generate database schema with schema spy, monitor file until doc is created and copy it
	mkdir -p database/schema/output && chmod a+w database/schema/output
	cd database/schema && $(COMPOSE) run -u 1001:1001 --rm schemaspy -configFile /config/schemaspy.properties -imageformat png
	sleep 10
	cp database/schema/output/diagrams/summary/relationships.real.large.png docs/static/img/mediator/schema.png
	cd database/schema && $(COMPOSE) down -v && rm -rf output

mock:
	mockgen -package mockdb -destination database/mock/store.go github.com/stacklok/mediator/internal/db Store
