#!/bin/bash -eu
# SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
# SPDX-License-Identifier: Apache-2.0

# Download dependency and let the Go package manager do all the work
printf "package main\nimport _ \"github.com/AdamKorcz/go-118-fuzz-build/testing\"\n" > ./cmd/cli/register.go
go mod tidy

# ClusterfuzzLite does not support different packages in the same directory,
#  and the jq package has its tests in a _test package.
#  We create a jq_test directory and move the tests there to make it work.
mkdir internal/engine/eval/jq/jq_test
mv internal/engine/eval/jq/fuzz_test.go internal/engine/eval/jq/jq_test/
compile_native_go_fuzzer github.com/mindersec/minder/internal/engine/eval/jq/jq_test FuzzJqEval FuzzJqEval
compile_native_go_fuzzer github.com/mindersec/minder/internal/engine/eval/rego FuzzRegoEval FuzzRegoEval
compile_native_go_fuzzer github.com/mindersec/minder/internal/providers/github/webhook FuzzGitHubEventParsers FuzzGitHubEventParsers
compile_native_go_fuzzer github.com/mindersec/minder/internal/engine/ingester/diff FuzzDiffParse FuzzDiffParse
compile_native_go_fuzzer github.com/mindersec/minder/internal/crypto FuzzEncryptDecrypt FuzzEncryptDecrypt
compile_native_go_fuzzer github.com/mindersec/minder/internal/auth/jwt FuzzParseAndValidate FuzzParseAndValidate
compile_native_go_fuzzer github.com/mindersec/minder/internal/util/cli FuzzRenderMarkdown FuzzRenderMarkdown
