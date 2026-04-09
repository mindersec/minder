// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestIsBundleError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "standard error containing lower case bundle",
			err:      errors.New("cannot delete rule type from bundle"),
			expected: true,
		},
		{
			name:     "grpc structured error containing upper case BUNDLE",
			err:      status.Errorf(codes.InvalidArgument, "cannot delete rule type from BUNDLE"),
			expected: true,
		},
		{
			name:     "unrelated error",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isBundleError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errStr   string
		expected []string
	}{
		{
			name:     "single profile",
			errStr:   "cannot delete: rule type is used by profiles foo",
			expected: []string{"foo"},
		},
		{
			name:     "multiple profiles",
			errStr:   "cannot delete: rule type is used by profiles foo, bar, baz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "no profiles mentioned",
			errStr:   "some other error",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractProfiles(tt.errStr)
			require.Equal(t, tt.expected, result)
		})
	}
}

// mockRuleTypeServiceClient is a simple mock for testing deleteRuleTypes without gomock
type mockRuleTypeServiceClient struct {
	minderv1.RuleTypeServiceClient
	deleteFunc func(ctx context.Context, in *minderv1.DeleteRuleTypeRequest, opts ...grpc.CallOption) (*minderv1.DeleteRuleTypeResponse, error)
}

func (m *mockRuleTypeServiceClient) DeleteRuleType(ctx context.Context, in *minderv1.DeleteRuleTypeRequest, opts ...grpc.CallOption) (*minderv1.DeleteRuleTypeResponse, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, in, opts...)
	}
	return &minderv1.DeleteRuleTypeResponse{}, nil
}

func TestDeleteRuleTypes_Categorization(t *testing.T) {
	t.Parallel()

	mockClient := &mockRuleTypeServiceClient{
		deleteFunc: func(ctx context.Context, in *minderv1.DeleteRuleTypeRequest, opts ...grpc.CallOption) (*minderv1.DeleteRuleTypeResponse, error) {
			switch in.GetId() {
			case "rule1":
				return &minderv1.DeleteRuleTypeResponse{}, nil
			case "rule2":
				return nil, status.Errorf(codes.InvalidArgument, "cannot delete rule type from bundle")
			case "rule3":
				return nil, status.Errorf(codes.FailedPrecondition, "cannot delete: rule type rule3 is used by profiles prof1, prof2")
			default:
				return nil, errors.New("unknown error")
			}
		},
	}

	rules := []*minderv1.RuleType{
		{Id: "rule1", Name: "successful_rule"},
		{Id: "rule2", Name: "bundle_rule"},
		{Id: "rule3", Name: "profile_rule"},
	}

	deleted, bundleBlocked, profileBlocked := deleteRuleTypes(context.Background(), mockClient, rules, "project_id")

	require.Len(t, deleted, 1)
	require.Equal(t, "successful_rule", deleted[0])

	require.Len(t, bundleBlocked, 1)
	require.Equal(t, "bundle_rule", bundleBlocked[0])

	require.Len(t, profileBlocked, 1)
	require.Equal(t, "profile_rule", profileBlocked[0].Name)
	require.Equal(t, []string{"prof1", "prof2"}, profileBlocked[0].Profiles)
}
