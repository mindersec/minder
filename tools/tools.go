// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

//go:generate go install github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go install mvdan.cc/gofumpt
//go:generate go install github.com/daixiang0/gci
//go:generate go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt
//go:generate go install golang.org/x/tools/cmd/goimports
//go:generate go install golang.org/x/lint/golint
//go:generate go install github.com/go-critic/go-critic/cmd/gocritic
//go:generate go install github.com/bufbuild/buf/cmd/buf
//go:generate go install github.com/norwoodj/helm-docs/cmd/helm-docs
//go:generate go install github.com/openfga/cli/cmd/fga
//go:generate go install github.com/mikefarah/yq/v4
//go:generate go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

// nolint

import (
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/daixiang0/gci"
	_ "github.com/go-critic/go-critic/cmd/gocritic"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "github.com/mikefarah/yq/v4"
	_ "github.com/norwoodj/helm-docs/cmd/helm-docs"
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	_ "github.com/openfga/cli/cmd/fga"
	_ "github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc"
	_ "go.uber.org/mock/mockgen"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/goimports"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "mvdan.cc/gofumpt"
)
