// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import "errors"

var (
	// ErrProviderGitBranchNotFound is returned when the branch is not found
	ErrProviderGitBranchNotFound = errors.New("branch not found")
	// ErrRepositoryEmpty is returned when the repository is empty
	ErrRepositoryEmpty = errors.New("repository is empty")
	// ErrRepositoryTooLarge is returned when the configured size limit is exceeded
	ErrRepositoryTooLarge = errors.New("repository is too large to clone")
)
