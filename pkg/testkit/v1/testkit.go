// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package v1 contains the v1 version of the testkit package. This is meant to
// aid developers in testing out minder rule types and other components.
package v1

import (
	"net/http"

	"github.com/mindersec/minder/internal/engine/ingester/git"
)

// TestKit implements a set of interfaces for testing
// purposes. e.g. for testing rule types.
type TestKit struct {
	ingestType string
	// gitDir is the directory where the git repository is cloned
	gitDir string

	// mockFS contains the filesystem representation for git ingestion testing
	mockFS map[string]string

	// HTTP
	httpHandler http.Handler
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

// WithMockFS is a functional option to configure the TestKit with a mocked filesystem
func WithMockFS(fs map[string]string) Option {
	return func(tk *TestKit) {
		tk.mockFS = fs
	}
}

// WithHandlerFunc is a functional option to configure the TestKit to use a specific Handler
func WithHandlerFunc(hf http.HandlerFunc) Option {
	return func(tk *TestKit) {
		tk.httpHandler = hf
	}
}

// WithHTTP is a functional option to set the HTTP response
func WithHTTP(status int, body []byte, headers map[string]string) Option {
	return WithHandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	})
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
