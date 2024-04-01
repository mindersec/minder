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

default: help

.PHONY: help
help: ## list makefile targets
	@echo "Usage: make [target] ...\n"
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##";} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST) | sort
