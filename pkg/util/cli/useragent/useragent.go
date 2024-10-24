// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package useragent contains utilities for setting up the CLI's user agent
package useragent

import (
	"fmt"
	"runtime"

	"github.com/mindersec/minder/internal/constants"
)

// GetUserAgent returns the user agent string for the CLI
// Note that the user agent used here replicates the user agent used by the
// browsers.
// e.g. <product>/<product-version> (<system-information>) <platform> (<platform-details>) <extensions>
//
// In our case, we'll leave extensions empty.
func GetUserAgent() string {
	product := "minder-cli"
	productVersion := constants.CLIVersion

	userAgent := fmt.Sprintf("%s/%s (%s) %s (%s)",
		product, productVersion, runtime.GOOS, runtime.GOARCH, runtime.Version())

	return userAgent
}
