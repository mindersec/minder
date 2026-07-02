// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package v1 contains the v1 version of the testkit package. This is meant to
// aid developers in testing out minder rule types and other components.
package v1

import (
	"net/http"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
)

// TestKit implements a set of interfaces for testing
// purposes. e.g. for testing rule types.
type TestKit struct {
	// gitFS is the filesystem used for git ingestion testing.
	// Both WithGitDir and WithGitFiles populate this field.
	gitFS billy.Filesystem

	// HTTP
	httpHandler http.Handler
}

// Option is a functional option type for TestKit
type Option func(*TestKit)

// WithGitDir is a functional option to set the git directory.
// It eagerly creates an osfs.Filesystem rooted at dir.
func WithGitDir(dir string) Option {
	return func(tk *TestKit) {
		tk.gitFS = osfs.New(dir)
	}
}

// WithGitFiles is a functional option to configure the TestKit with a
// mocked in-memory filesystem for git ingestion testing. Each key is
// a file path and each value is the file content.
func WithGitFiles(files map[string]string) Option {
	return func(tk *TestKit) {
		fs := memfs.New()
		for path, content := range files {
			dir := filepath.Dir(path)
			if dir != "" && dir != "." {
				_ = fs.MkdirAll(dir, 0755)
			}
			f, err := fs.Create(path)
			if err != nil {
				continue
			}
			_, _ = f.Write([]byte(content))
			_ = f.Close()
		}
		tk.gitFS = fs
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
	pt := &TestKit{}

	for _, opt := range opts {
		opt(pt)
	}

	return pt
}
