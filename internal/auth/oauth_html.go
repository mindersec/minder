// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import _ "embed"

// OAuthSuccessHtml is the html page sent to the client upon successful enrollment
// via CLI
//
//go:embed html/oauth_success.html
var OAuthSuccessHtml []byte
