// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestGetAllRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		profile  *Profile
		expected int
	}{
		{
			name:     "nil profile fields",
			profile:  &Profile{},
			expected: 0,
		},
		{
			name: "rules in single entity type",
			profile: &Profile{
				Repository: []*Profile_Rule{
					{Type: "rule-a"},
					{Type: "rule-b"},
				},
			},
			expected: 2,
		},
		{
			name: "rules across multiple entity types",
			profile: &Profile{
				Repository:  []*Profile_Rule{{Type: "rule-a"}},
				Artifact:    []*Profile_Rule{{Type: "rule-b"}},
				PullRequest: []*Profile_Rule{{Type: "rule-c"}},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rules := tt.profile.GetAllRules()
			assert.Len(t, rules, tt.expected)
		})
	}
}

func TestGetAllRulesCoversAllRuleFields(t *testing.T) {
	t.Parallel()

	p := &Profile{}
	fields := p.ProtoReflect().Descriptor().Fields()
	want := 0
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if fd.IsList() && fd.Message() != nil &&
			fd.Message().FullName() == (&Profile_Rule{}).ProtoReflect().Descriptor().FullName() {
			list := p.ProtoReflect().Mutable(fd).List()
			list.Append(protoreflect.ValueOfMessage((&Profile_Rule{Type: string(fd.Name())}).ProtoReflect()))
			want++
		}
	}

	assert.Len(t, p.GetAllRules(), want)
}
