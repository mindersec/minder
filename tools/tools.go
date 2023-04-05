//go:build tools

package tools

//go:generate go install github.com/golangci/golangci-lint/cmd/golangci-lint
//go:generate go install mvdan.cc/gofumpt
//go:generate go install github.com/daixiang0/gci
//go:generate go install github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt
//go:generate go install golang.org/x/tools/cmd/goimports
//go:generate go install golang.org/x/lint/golint
//go:generate go install github.com/go-critic/go-critic/cmd/gocritic

// nolint

import (
	_ "github.com/daixiang0/gci"
	_ "github.com/go-critic/go-critic/cmd/gocritic"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/goimports"
	_ "mvdan.cc/gofumpt"
)
