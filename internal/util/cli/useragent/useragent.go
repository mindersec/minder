//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package useragent contains utilities for setting up the CLI's user agent
package useragent

import (
	"fmt"
	"runtime"

	"github.com/stacklok/minder/internal/constants"
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
