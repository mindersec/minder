// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package main provides the entrypoint for the rule development cli
package main

import (
	"github.com/mindersec/minder/cmd/dev/app"
	_ "github.com/mindersec/minder/cmd/dev/app/rule_type"
)

func main() {
	app.Execute()
}
