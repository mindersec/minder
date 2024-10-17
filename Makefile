# SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

default: help

SHELL       ?= /bin/bash
.SHELLFLAGS ?= -ec

projectname?=minder

# Include all the makefiles
include examples/Makefile
include .mk/gen.mk
include .mk/db.mk
include .mk/identity.mk
include .mk/test.mk
include .mk/helm.mk
include .mk/develop.mk
include .mk/build.mk
include .mk/authz.mk


# OS detection for docker compose
OS := $(shell uname -s)

COMPOSE?=docker compose
CONTAINER?=docker

# Services to run in docker compose. Defaults to all
services?=

# Arguments to pass to docker compose
COMPOSE_ARGS?=-d

# Flags and env vars for ko
KO_DOCKER_REPO?=ko.local
KO_PUSH_IMAGE?=false
KO_PLATFORMS=linux/amd64,linux/arm64

# Helm package version
HELM_PACKAGE_VERSION?=0.1.0

.PHONY: help
help: ## list makefile targets
	@echo "Usage: make [target] ..."
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##";} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST) | sort
