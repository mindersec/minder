#!/bin/bash -eu
# Copyright 2024 Stacklok, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
################################################################################

# Download dependency and let the Go package manager do all the work
printf "package main\nimport _ \"github.com/AdamKorcz/go-118-fuzz-build/testing\"\n" > ./cmd/cli/register.go
go mod tidy

# ClusterfuzzLite does not support different packages in the same directory,
#  and the jq package has its tests in a _test package.
#  We create a jq_test directory and move the tests there to make it work.
mkdir internal/engine/eval/jq/jq_test
mv internal/engine/eval/jq/fuzz_test.go internal/engine/eval/jq/jq_test/
compile_native_go_fuzzer github.com/stacklok/minder/internal/engine/eval/jq/jq_test FuzzJqEval FuzzJqEval
compile_native_go_fuzzer github.com/stacklok/minder/internal/engine/eval/rego FuzzRegoEval FuzzRegoEval
compile_native_go_fuzzer github.com/stacklok/minder/internal/controlplane FuzzGitHubEventParsers FuzzGitHubEventParsers
compile_native_go_fuzzer github.com/stacklok/minder/internal/engine/ingester/diff FuzzDiffParse FuzzDiffParse
compile_native_go_fuzzer github.com/stacklok/minder/internal/crypto FuzzEncryptDecrypt FuzzEncryptDecrypt
compile_native_go_fuzzer github.com/stacklok/minder/internal/auth/jwt FuzzParseAndValidate FuzzParseAndValidate
