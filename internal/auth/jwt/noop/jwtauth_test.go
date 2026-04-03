// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package noop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopJwtValidator_ParseAndValidate(t *testing.T) {
	t.Parallel()

	expectedSubject := "test-subject"
	validator := NewJwtValidator(expectedSubject)

	tok, err := validator.ParseAndValidate("any-token")

	require.NoError(t, err)
	assert.NotNil(t, tok)
	assert.Equal(t, expectedSubject, tok.Subject())
}
