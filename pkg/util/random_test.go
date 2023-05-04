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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRandomInt(t *testing.T) {
	min := int64(1)
	max := int64(10)
	randomInt := RandomInt(min, max)
	require.GreaterOrEqual(t, randomInt, min)
	require.LessOrEqual(t, randomInt, max)
}

func TestRandomString(t *testing.T) {
	randomString := RandomString(10)
	require.NotEmpty(t, randomString)
	require.Len(t, randomString, 10)
}

func TestRandomEmail(t *testing.T) {
	email := RandomEmail()
	require.NotEmpty(t, email)
	require.Contains(t, email, "@")
	require.Contains(t, email, ".")
	require.Len(t, email, 22)
}

func TestRandomName(t *testing.T) {
	name := RandomName()
	require.NotEmpty(t, name)
	require.Len(t, name, 10)
}
