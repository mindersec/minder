// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package assets contains the static assets.
package assets

import (
	"embed"
)

// StaticAssets are the static assets, such as images.
//
//go:embed static
var StaticAssets embed.FS
