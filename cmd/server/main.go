// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package main provides the entrypoint for the minder server
package main

import "github.com/mindersec/minder/cmd/server/app"

func main() {
	app.Execute()
}
