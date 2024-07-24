// Copyright 2024 Stacklok, Inc.
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
// Package rule provides the CLI subcommand for managing rules

package artifact

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func TestBuildFilter(t *testing.T) {
	t.Parallel()
	simpleRegex := `^simpleregex$`
	compiledSimpleRegex := regexp.MustCompile(simpleRegex)
	for _, tc := range []struct {
		name     string
		tags     []string
		regex    string
		expected *filter
		mustErr  bool
	}{
		{
			name:    "tags",
			tags:    []string{"hello", "bye"},
			mustErr: false,
			expected: &filter{
				tagMatcher:      &tagListMatcher{tags: []string{"hello", "bye"}},
				retentionPeriod: time.Time{},
			},
		},
		{
			name:    "empty-tag",
			tags:    []string{"hello", ""},
			mustErr: true,
		},
		{
			name:    "regex",
			tags:    []string{},
			regex:   simpleRegex,
			mustErr: false,
			expected: &filter{
				tagMatcher: &tagRegexMatcher{
					re: compiledSimpleRegex,
				},
			},
		},
		{
			name:    "invalidregexp",
			tags:    []string{},
			regex:   `$(invalid^`,
			mustErr: true,
		},
		{
			name:    "valid-long-regexp",
			tags:    []string{},
			regex:   `^` + strings.Repeat("A", 1000) + `$`,
			mustErr: true,
		},
		{
			name:    "no-tags",
			tags:    []string{},
			mustErr: false,
			expected: &filter{
				tagMatcher: &tagAllMatcher{},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f, err := BuildFilter(tc.tags, tc.regex)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, f.tagMatcher)
			require.Equal(t, tc.expected.tagMatcher, f.tagMatcher)
		})
	}
}

func Test_filter_IsSkippable(t *testing.T) {
	t.Parallel()

	type fieldsArgs struct {
		tags     []string
		tagRegex string
	}
	type args struct {
		createdAt time.Time
		tags      []string
	}
	tests := []struct {
		name         string
		fieldsArgs   fieldsArgs
		args         args
		wantBuildErr bool
		wantSkip     bool
	}{
		{
			name: "tags and regex is not allowed",
			fieldsArgs: fieldsArgs{
				tags:     []string{"tag1", "tag2"},
				tagRegex: "tag.*",
			},
			args:         args{},
			wantBuildErr: true,
		},
		{
			name: "empty tag is not allowed",
			fieldsArgs: fieldsArgs{
				tags:     []string{""},
				tagRegex: "",
			},
			args:         args{},
			wantBuildErr: true,
		},
		{
			name: "invalid regex is not allowed",
			fieldsArgs: fieldsArgs{
				tags:     nil,
				tagRegex: "[",
			},
			args:         args{},
			wantBuildErr: true,
		},
		{
			name: "no tags specified, match all",
			fieldsArgs: fieldsArgs{
				tags:     nil,
				tagRegex: "",
			},
			args: args{
				createdAt: time.Now(),
				tags:      []string{"tag1", "tag2"},
			},
		},
		{
			name: "artifact is older than retention period",
			fieldsArgs: fieldsArgs{
				tags:     nil,
				tagRegex: "",
			},
			args: args{
				createdAt: provifv1.ArtifactTypeContainerRetentionPeriod.AddDate(0, 0, -1),
				tags:      []string{"tag1", "tag2"},
			},
			wantSkip: true,
		},
		{
			name: "artifact has no tags",
			fieldsArgs: fieldsArgs{
				tags:     nil,
				tagRegex: "",
			},
			args: args{
				createdAt: time.Now(),
				tags:      nil,
			},
			wantSkip: true,
		},
		{
			name: "artifact has empty tag",
			fieldsArgs: fieldsArgs{
				tags:     nil,
				tagRegex: "",
			},
			args: args{
				createdAt: time.Now(),
				tags:      []string{""},
			},
			wantSkip: true,
		},
		{
			name: "artifact is a signature",
			fieldsArgs: fieldsArgs{
				tags:     nil,
				tagRegex: "",
			},
			args: args{
				createdAt: time.Now(),
				tags:      []string{".sig"},
			},
			wantSkip: true,
		},
		{
			name: "artifact tags does not match",
			fieldsArgs: fieldsArgs{
				tags:     []string{"tag1", "tag2"},
				tagRegex: "",
			},
			args: args{
				createdAt: time.Now(),
				tags:      []string{"tag3", "tag4"},
			},
			wantSkip: true,
		},
		{
			name: "artifact tags does match",
			fieldsArgs: fieldsArgs{
				tags:     []string{"tag1", "tag2"},
				tagRegex: "",
			},
			args: args{
				createdAt: time.Now(),
				tags:      []string{"tag1", "tag2"},
			},
		},
		{
			name: "artifact tag subset does match",
			fieldsArgs: fieldsArgs{
				tags:     []string{"tag1"},
				tagRegex: "",
			},
			args: args{
				createdAt: time.Now(),
				tags:      []string{"tag1", "tag2"},
			},
		},
		{
			name: "artifact tag superset does not match",
			fieldsArgs: fieldsArgs{
				tags:     []string{"tag1", "tag2"},
				tagRegex: "",
			},
			args: args{
				createdAt: time.Now(),
				tags:      []string{"tag1"},
			},
			wantSkip: true,
		},
		{
			name: "artifact tags does match with regex",
			fieldsArgs: fieldsArgs{
				tags:     nil,
				tagRegex: "tag.*",
			},
			args: args{
				createdAt: time.Now(),
				tags:      []string{"tag1", "tag2"},
			},
		},
		{
			name: "artifact tags does not match with regex",
			fieldsArgs: fieldsArgs{
				tags:     nil,
				tagRegex: "tag.*",
			},
			args: args{
				createdAt: time.Now(),
				tags:      []string{"teg3", "teg4"},
			},
			wantSkip: true,
		},
		{
			name: "artifact with no tags does not match with regex",
			fieldsArgs: fieldsArgs{
				tags:     nil,
				tagRegex: "tag.*",
			},
			args: args{
				createdAt: time.Now(),
				tags:      nil,
			},
			wantSkip: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, err := BuildFilter(tt.fieldsArgs.tags, tt.fieldsArgs.tagRegex)
			if tt.wantBuildErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if err := f.IsSkippable(tt.args.createdAt, tt.args.tags); tt.wantSkip {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}
