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

# exclude auto-generated DB code as well as mocks
# in future, we may want to parse these from a file instead of hardcoding them
# in the Makefile
COVERAGE_EXCLUSIONS="internal/db\|/mock/"
COVERAGE_PACKAGES=./internal/...

.PHONY: clean
clean: ## clean up environment
	rm -rf dist/* & rm -rf bin/*

.PHONY: test
test: clean init-examples ## run tests in verbose mode
	go test -json -race -v ./... | gotestfmt

.PHONY: test-silent
test-silent: clean init-examples ## run tests in a silent mode (errors only output)
	go test -json -race -v ./... | gotestfmt -hide "all"

.PHONY: cover
cover: init-examples ## display test coverage
	# as of the time of writing, there is a bug in the new coverage logic
	# implemented in go 1.22. The recommended workaround is to disable the new
	# coverage logic until this is fixed.
	# See: https://github.com/golang/go/issues/65653
	GOEXPERIMENT=nocoverageredesign go test -v -coverpkg=${COVERAGE_PACKAGES} -coverprofile=coverage.out.tmp -race ./...
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
