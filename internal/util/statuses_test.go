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

package util_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/stacklok/minder/internal/util"
)

func TestNiceStatusCreation(t *testing.T) {
	t.Parallel()

	s := util.GetNiceStatus(codes.OK)
	require.Equal(t, codes.OK, s.Code)
	require.Equal(t, "OK", s.Name)
	require.Equal(t, "OK", s.Description)
	require.Equal(t, "OK is returned on success.", s.Details)

	expected := "Code: 0\nName: OK\nDescription: OK\nDetails: OK is returned on success."
	require.Equal(t, expected, fmt.Sprint(s))
}
