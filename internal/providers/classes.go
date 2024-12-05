// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package providers

import "github.com/mindersec/minder/internal/db"

// ListProviderClasses returns a list of provider classes.
func ListProviderClasses() []string {
	return []string{
		string(db.ProviderClassGithub),
		string(db.ProviderClassGithubApp),
		string(db.ProviderClassDockerhub),
		string(db.ProviderClassGhcr),
	}
}
