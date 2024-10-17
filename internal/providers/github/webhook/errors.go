// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"errors"
	"fmt"
)

// errRepoNotFound is returned when a repository is not found
var errRepoNotFound = errors.New("repository not found")

// errArtifactNotFound is returned when an artifact is not found
var errArtifactNotFound = errors.New("artifact not found")

// errArtifactVersionSkipped is returned when an artifact is skipped because it has no tags
var errArtifactVersionSkipped = errors.New("artifact version skipped, has no tags")

// errRepoIsPrivate is returned when a repository is private
var errRepoIsPrivate = errors.New("repository is private")

// errNotHandled is returned when a webhook event is not handled
var errNotHandled = errors.New("webhook event not handled")

// newErrNotHandled returns a new errNotHandled error
func newErrNotHandled(smft string, args ...any) error {
	msg := fmt.Sprintf(smft, args...)
	return fmt.Errorf("%w: %s", errNotHandled, msg)
}
