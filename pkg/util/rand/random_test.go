// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rand_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/util/rand"
)

func TestRandomInt(t *testing.T) {
	t.Parallel()

	min := int64(1)
	max := int64(10)
	seed := int64(12345)
	randomInt := rand.RandomInt(min, max, seed)
	require.GreaterOrEqual(t, randomInt, min)
	require.LessOrEqual(t, randomInt, max)
}

func TestRandomString(t *testing.T) {
	t.Parallel()
	seed := int64(12345)
	randomString := rand.RandomString(10, seed)
	require.NotEmpty(t, randomString)
	require.Len(t, randomString, 10)
}

func TestRandomName(t *testing.T) {
	t.Parallel()

	seed := int64(12345)
	name := rand.RandomName(seed)
	require.NotEmpty(t, name)
	require.Len(t, name, 10)
}

func TestRandomFrom(t *testing.T) {
	t.Parallel()

	seed := int64(12345)
	choices := []string{"a", "b", "c", "d", "e"}
	choice := rand.RandomFrom(choices, seed)
	require.NotEmpty(t, choice)
	require.Contains(t, choices, choice)
}
