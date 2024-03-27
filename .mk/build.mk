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

.PHONY: build
build: build-minder-cli build-minder-server build-mindev build-reminder-server ## build all binaries


.PHONY: build-mindev
build-mindev: ## build mindev golang binary
	@echo "Building mindev..."
	@CGO_ENABLED=0 go build -trimpath -tags '$(BUILDTAGS)' -o ./bin/mindev ./cmd/dev

.PHONY: build-minder-cli
build-minder-cli: ## build minder cli
	@echo "Building $(projectname)..."
	@CGO_ENABLED=0 go build \
		-trimpath \
		-tags '$(BUILDTAGS)' \
		-ldflags "-X github.com/stacklok/minder/internal/constants.CLIVersion=$(shell git describe --abbrev=0 --tags)+ref.$(shell git rev-parse --short HEAD)" \
		-o ./bin/$(projectname) ./cmd/cli

.PHONY: build-minder-server
build-minder-server: ## build minder-server
	@echo "Building $(projectname)-server..."
	@CGO_ENABLED=0 go build -trimpath -tags '$(BUILDTAGS)' -o ./bin/$(projectname)-server ./cmd/server

.PHONY: build-reminder-server
build-reminder-server: ## build reminder server
	@echo "Building reminder..."
	@CGO_ENABLED=0 go build -trimpath -tags '$(BUILDTAGS)' -o ./bin/reminder ./cmd/reminder
