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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

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
