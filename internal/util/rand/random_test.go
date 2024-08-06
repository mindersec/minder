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

package rand_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/rand"
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
