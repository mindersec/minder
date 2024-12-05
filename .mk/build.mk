# SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

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
		-ldflags "-X github.com/mindersec/minder/internal/constants.CLIVersion=$(shell git describe --abbrev=0 --tags)+ref.$(shell git rev-parse --short HEAD)" \
		-o ./bin/$(projectname) ./cmd/cli

.PHONY: build-minder-server
build-minder-server: ## build minder-server
	@echo "Building $(projectname)-server..."
	@CGO_ENABLED=0 go build -trimpath -tags '$(BUILDTAGS)' -o ./bin/$(projectname)-server ./cmd/server

.PHONY: build-reminder-server
build-reminder-server: ## build reminder server
	@echo "Building reminder..."
	@CGO_ENABLED=0 go build -trimpath -tags '$(BUILDTAGS)' -o ./bin/reminder ./cmd/reminder
