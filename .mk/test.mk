# SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

# exclude auto-generated DB code as well as mocks
# in future, we may want to parse these from a file instead of hardcoding them
# in the Makefile
COVERAGE_EXCLUSIONS="internal/db\|/mock/\|internal/auth/keycloak/client\|internal/proto\|pkg/api\|pkg/testkit"
COVERAGE_PACKAGES=./internal/...,./pkg/...

.PHONY: clean
clean:: ## clean up environment
	rm -rf dist/* && rm -rf bin/*

.PHONY: test
test: clean init-examples ## run tests in verbose mode
	go test -json -race -v ./... | gotestfmt

.PHONY: test-silent
test-silent: clean init-examples ## run tests in a silent mode (errors only output)
	go test -json -race -v ./... | gotestfmt -hide "all"

.PHONY: cover
cover: init-examples ## display test coverage
	go test -v -coverpkg=${COVERAGE_PACKAGES} -coverprofile=coverage.out.tmp -race ./...
	cat coverage.out.tmp | grep -v ${COVERAGE_EXCLUSIONS} > coverage.out
	rm coverage.out.tmp
	go tool cover -func=coverage.out

.PHONY: test-cover-silent
test-cover-silent: clean init-examples  ## Run test coverage in a silent mode (errors only output)
	go test -json -race -v -coverpkg=${COVERAGE_PACKAGES} -coverprofile=coverage.out.tmp ./... | gotestfmt -hide "all"
	cat coverage.out.tmp | grep -v ${COVERAGE_EXCLUSIONS} > coverage.out
	rm coverage.out.tmp
	go tool cover -func=coverage.out

.PHONY: lint
lint: lint-go lint-buf ## lint

.PHONY: lint-go
lint-go:
	golangci-lint run

.PHONY: lint-buf
lint-buf:
	buf lint
