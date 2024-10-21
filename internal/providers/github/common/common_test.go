// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package common provides common utilities for the GitHub provider
package common

import (
	"testing"

	go_github "github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/providers/github/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestConvertRepositories(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		repos    []*go_github.Repository
		expected []*minderv1.Repository
	}{
		{
			name: "Convert non-archived github repositories to minder repositories",
			repos: []*go_github.Repository{
				{
					Name:           go_github.String("minder"),
					Owner:          &go_github.User{Login: go_github.String("owner")},
					ID:             go_github.Int64(12345),
					FullName:       go_github.String("mindersec/minder"),
					HTMLURL:        go_github.String("https://github.com/mindersec/minder"),
					HooksURL:       go_github.String("https://github.com/mindersec/minder/hooks"),
					DeploymentsURL: go_github.String("https://github.com/mindersec/minder/deploy"),
					CloneURL:       go_github.String("https://github.com/mindersec/minder/clone"),
					Private:        go_github.Bool(false),
					Fork:           go_github.Bool(false),
					Archived:       go_github.Bool(false),
				},
			},
			expected: []*minderv1.Repository{
				{
					Name:      "minder",
					Owner:     "owner",
					RepoId:    12345,
					HookUrl:   "https://github.com/mindersec/minder/hooks",
					DeployUrl: "https://github.com/mindersec/minder/deploy",
					CloneUrl:  "https://github.com/mindersec/minder/clone",
					IsPrivate: false,
					IsFork:    false,
					Properties: gitHubRepoToMap(&go_github.Repository{
						Name:           go_github.String("minder"),
						Owner:          &go_github.User{Login: go_github.String("owner")},
						ID:             go_github.Int64(12345),
						FullName:       go_github.String("mindersec/minder"),
						HTMLURL:        go_github.String("https://github.com/mindersec/minder"),
						HooksURL:       go_github.String("https://github.com/mindersec/minder/hooks"),
						DeploymentsURL: go_github.String("https://github.com/mindersec/minder/deploy"),
						CloneURL:       go_github.String("https://github.com/mindersec/minder/clone"),
						Private:        go_github.Bool(false),
						Fork:           go_github.Bool(false),
						Archived:       go_github.Bool(false),
					}),
				},
			},
		},
		{
			name: "Skip archived github repositories",
			repos: []*go_github.Repository{
				{
					Name:           go_github.String("feedback"),
					Owner:          &go_github.User{Login: go_github.String("owner")},
					ID:             go_github.Int64(2),
					HooksURL:       go_github.String("https://github.com/stacklok/feedback/hooks"),
					DeploymentsURL: go_github.String("https://github.com/stacklok/feedback/deploy"),
					CloneURL:       go_github.String("https://github.com/stacklok/feedback/clone"),
					Private:        go_github.Bool(false),
					Fork:           go_github.Bool(false),
					Archived:       go_github.Bool(true),
				},
			},
			expected: []*minderv1.Repository(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := ConvertRepositories(tt.repos)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// function to get properties.GitHubRepoToMap as structpb.Struct
func gitHubRepoToMap(repo *go_github.Repository) *structpb.Struct {
	propsMap := properties.GitHubRepoToMap(repo)
	props, err := structpb.NewStruct(propsMap)
	if err != nil {
		panic(err)
	}
	return props
}
