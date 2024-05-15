// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package trusty provides an evaluator that uses the trusty API
package trusty

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewSummaryPrHandler(t *testing.T) {
	t.Parallel()

	// newSummaryPrHandler must never fail. The only failure point
	// right now is the pr comment template
	_, err := newSummaryPrHandler(&v1.PullRequest{}, nil, "")
	require.NoError(t, err)
}
