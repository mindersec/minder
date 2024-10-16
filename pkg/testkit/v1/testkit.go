// Copyright 2024 Stacklok, Inc
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

// Package v1 contains the v1 version of the testkit package. This is meant to
// aid developers in testing out minder rule types and other components.
package v1

import (
	"net/http/httptest"

	"github.com/mindersec/minder/internal/engine/ingester/git"
)

// TestKit implements a set of interfaces for testing
// purposes. e.g. for testing rule types.
type TestKit struct {
	ingestType string
	// gitDir is the directory where the git repository is cloned
	gitDir string

	// HTTP
	httpRecorder *httptest.ResponseRecorder
	httpStatus   int
	httpBody     []byte
	httpHeaders  map[string]string
}

// Option is a functional option type for TestKit
type Option func(*TestKit)

// WithGitDir is a functional option to set the git directory
// Note that if the `git` ingest type is used, you need to overwrite the
// ingester in the rule type engine.
func WithGitDir(dir string) Option {
	return func(tp *TestKit) {
		tp.ingestType = git.GitRuleDataIngestType
		tp.gitDir = dir
	}
}

// WithHTTP is a functional option to set the HTTP response
func WithHTTP(status int, body []byte, headers map[string]string) Option {
	return func(tp *TestKit) {
		tp.httpRecorder = httptest.NewRecorder()
		tp.httpStatus = status
		tp.httpBody = body
		tp.httpHeaders = headers
	}
}

// NewTestKit creates a new TestKit
func NewTestKit(opts ...Option) *TestKit {
	pt := &TestKit{
		gitDir: ".",
	}

	for _, opt := range opts {
		opt(pt)
	}

	return pt
}
