// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhooksecret

import (
	sum "crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	base := "baseString"
	uniq := "uniqueString"
	expectedHash := sum.New()
	expectedHash.Write([]byte(base + uniq))
	expectedSecret := hex.EncodeToString(expectedHash.Sum(nil))

	secret, err := New(base, uniq)
	assert.NoError(t, err)
	assert.Equal(t, expectedSecret, secret, "they should be equal")
}

func TestNew_EmptyStrings(t *testing.T) {
	t.Parallel()

	base := ""
	uniq := ""
	secret, err := New(base, uniq)
	assert.Error(t, err)
	assert.Equal(t, ErrEmptyBaseOrUniq, err)
	assert.Empty(t, secret)
}

func TestNew_SpecialCharacters(t *testing.T) {
	t.Parallel()

	base := "base@String!"
	uniq := "unique#String$"
	expectedHash := sum.New()
	expectedHash.Write([]byte(base + uniq))
	expectedSecret := hex.EncodeToString(expectedHash.Sum(nil))

	secret, err := New(base, uniq)
	assert.NoError(t, err)
	assert.Equal(t, expectedSecret, secret, "they should be equal")
}

func TestVerify(t *testing.T) {
	t.Parallel()

	base := "baseString"
	uniq := "uniqueString"
	secret, err := New(base, uniq)
	assert.NoError(t, err)

	assert.True(t, Verify(base, uniq, secret), "the secret should be valid")
	assert.False(t, Verify(base, uniq, "invalidSecret"), "the secret should be invalid")
}

func TestVerify_EmptyStrings(t *testing.T) {
	t.Parallel()

	base := ""
	uniq := ""
	secret, err := New(base, uniq)
	assert.Error(t, err)
	assert.False(t, Verify(base, uniq, secret), "the secret should be invalid")
}

func TestVerify_SpecialCharacters(t *testing.T) {
	t.Parallel()

	base := "base@String!"
	uniq := "unique#String$"
	secret, err := New(base, uniq)
	assert.NoError(t, err)

	assert.True(t, Verify(base, uniq, secret), "the secret should be valid")
	assert.False(t, Verify(base, uniq, "invalidSecret"), "the secret should be invalid")
}

func TestVerify_DifferentBaseUniq(t *testing.T) {
	t.Parallel()

	base1 := "baseString1"
	uniq1 := "uniqueString1"
	secret1, err := New(base1, uniq1)
	assert.NoError(t, err)

	base2 := "baseString2"
	uniq2 := "uniqueString2"
	secret2, err := New(base2, uniq2)
	assert.NoError(t, err)

	assert.False(t, Verify(base1, uniq1, secret2), "the secret should be invalid")
	assert.False(t, Verify(base2, uniq2, secret1), "the secret should be invalid")
}
