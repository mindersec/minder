// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/util/ptr"
	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestProtoValidationInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     proto.Message
		errMsg  string
		errCode codes.Code
	}{
		{
			name: "valid request",
			req: &v1.GetProviderRequest{
				Context: &v1.Context{
					Project: ptr.Ptr(uuid.New().String()),
				},
				Name: "valid-name",
			},
		},
		{
			name: "invalid request",
			req: &v1.GetProviderRequest{
				Context: &v1.Context{
					Project: ptr.Ptr(uuid.New().String()),
				},
				Name: "-?invalid",
			},
			errMsg:  "Validation failed:\n- Field 'name': value does not match regex pattern",
			errCode: codes.InvalidArgument,
		},
		{
			name: "invalid request with nested field",
			req: &v1.ListEvaluationResultsRequest{
				Context: &v1.Context{
					Project: ptr.Ptr(uuid.New().String()),
				},
				Entity: []*v1.EntityTypedId{
					{
						Id: "invalid-id",
					},
				},
			},
			errMsg:  "Validation failed:\n- Field 'entity[0].id': value must be a valid UUID",
			errCode: codes.InvalidArgument,
		},
	}

	validator, err := NewValidator()
	require.NoError(t, err)

	interceptor := ProtoValidationInterceptor(validator)

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := func(_ context.Context, _ interface{}) (interface{}, error) {
				return "response", nil
			}
			resp, err := interceptor(context.Background(), tt.req, nil, handler)
			if tt.errMsg != "" {
				require.Error(t, err)
				require.Nil(t, resp)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.errCode, st.Code())
				require.Contains(t, st.Message(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "response", resp)
		})
	}
}
