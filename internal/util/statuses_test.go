//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/util"
)

func TestNiceStatusCreation(t *testing.T) {
	t.Parallel()

	s := util.GetNiceStatus(codes.OK)
	require.Equal(t, codes.OK, s.Code)
	require.Equal(t, "OK", s.Name)
	require.Equal(t, "OK", s.Description)
	require.Equal(t, "OK is returned on success.", s.Details)

	expected := "Code: 0\nName: OK\nDescription: OK\nDetails: OK is returned on success."
	require.Equal(t, expected, fmt.Sprint(s))
}

func TestSanitizingInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		handler   grpc.UnaryHandler
		wantErr   bool
		errIsNice bool
	}{
		{
			name: "success",
			handler: func(_ context.Context, _ interface{}) (interface{}, error) {
				return "success", nil
			},
			wantErr: false,
		},
		{
			name: "some error",
			handler: func(_ context.Context, _ interface{}) (interface{}, error) {
				return nil, status.Error(codes.Internal, "some error")
			},
			wantErr: true,
		},
		{
			name: "nice error",
			handler: func(_ context.Context, _ interface{}) (interface{}, error) {
				return nil, util.UserVisibleError(codes.Internal, "some error")
			},
			wantErr:   true,
			errIsNice: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			i := util.SanitizingInterceptor()
			ret, err := i(ctx, nil, nil, tt.handler)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, ret)

				if tt.errIsNice {
					require.IsType(t, &util.NiceStatus{}, err)
					require.Contains(t, err.Error(), "Code: 13\nName: INTERNAL\nDescription: Server error\nDetails: some error")

					st := status.Convert(err)
					require.Equal(t, codes.Internal, st.Code())
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, ret)
		})
	}
}

func TestNiceStatusFromRpcError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    *status.Status
		want *util.NiceStatus
	}{
		{
			name: "OK",
			s:    status.New(codes.OK, "OK"),
			want: &util.NiceStatus{
				Code:        codes.OK,
				Name:        "OK",
				Description: "OK",
			},
		},
		{
			name: "CANCELLED",
			s:    status.New(codes.Canceled, "Cancelled"),
			want: &util.NiceStatus{
				Code:        codes.Canceled,
				Name:        "CANCELLED",
				Description: "Cancelled",
			},
		},
		{
			name: "UNKNOWN",
			s:    status.New(codes.Unknown, ""),
			want: &util.NiceStatus{
				Code:        codes.Unknown,
				Name:        "UNKNOWN",
				Description: "Unknown",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ns := util.FromRpcError(tt.s)
			require.Equal(t, tt.want.Code, ns.Code)
			require.Equal(t, tt.want.Name, ns.Name)
			require.Equal(t, tt.want.Description, ns.Description)
		})
	}
}

func TestSetCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    codes.Code
		results util.NiceStatus
	}{
		{
			name: "OK",
			code: codes.OK,
			results: util.NiceStatus{
				Code:        codes.OK,
				Name:        "OK",
				Description: "OK",
				// NOTE: let's not test the details here, as it's a bit too verbose
			},
		},
		{
			name: "CANCELLED",
			code: codes.Canceled,
			results: util.NiceStatus{
				Code:        codes.Canceled,
				Name:        "CANCELLED",
				Description: "Cancelled",
			},
		},
		{
			name: "UNKNOWN",
			code: codes.Unknown,
			results: util.NiceStatus{
				Code:        codes.Unknown,
				Name:        "UNKNOWN",
				Description: "Unknown",
			},
		},
		{
			name: "INVALID_ARGUMENT",
			code: codes.InvalidArgument,
			results: util.NiceStatus{
				Code:        codes.InvalidArgument,
				Name:        "INVALID_ARGUMENT",
				Description: "Invalid argument",
			},
		},
		{
			name: "DEADLINE_EXCEEDED",
			code: codes.DeadlineExceeded,
			results: util.NiceStatus{
				Code:        codes.DeadlineExceeded,
				Name:        "DEADLINE_EXCEEDED",
				Description: "Deadline exceeded",
			},
		},
		{
			name: "NOT_FOUND",
			code: codes.NotFound,
			results: util.NiceStatus{
				Code:        codes.NotFound,
				Name:        "NOT_FOUND",
				Description: "Not found",
			},
		},
		{
			name: "ALREADY_EXISTS",
			code: codes.AlreadyExists,
			results: util.NiceStatus{
				Code:        codes.AlreadyExists,
				Name:        "ALREADY_EXISTS",
				Description: "Already exists",
			},
		},
		{
			name: "PERMISSION_DENIED",
			code: codes.PermissionDenied,
			results: util.NiceStatus{
				Code:        codes.PermissionDenied,
				Name:        "PERMISSION_DENIED",
				Description: "Permission denied",
			},
		},
		{
			name: "RESOURCE_EXHAUSTED",
			code: codes.ResourceExhausted,
			results: util.NiceStatus{
				Code:        codes.ResourceExhausted,
				Name:        "RESOURCE_EXHAUSTED",
				Description: "Resource exhausted",
			},
		},
		{
			name: "FAILED_PRECONDITION",
			code: codes.FailedPrecondition,
			results: util.NiceStatus{
				Code:        codes.FailedPrecondition,
				Name:        "FAILED_PRECONDITION",
				Description: "Failed precondition",
			},
		},
		{
			name: "ABORTED",
			code: codes.Aborted,
			results: util.NiceStatus{
				Code:        codes.Aborted,
				Name:        "ABORTED",
				Description: "Aborted",
			},
		},
		{
			name: "OUT_OF_RANGE",
			code: codes.OutOfRange,
			results: util.NiceStatus{
				Code:        codes.OutOfRange,
				Name:        "OUT_OF_RANGE",
				Description: "Out of range",
			},
		},
		{
			name: "UNIMPLEMENTED",
			code: codes.Unimplemented,
			results: util.NiceStatus{
				Code:        codes.Unimplemented,
				Name:        "UNIMPLEMENTED",
				Description: "Unimplemented",
			},
		},
		{
			name: "INTERNAL",
			code: codes.Internal,
			results: util.NiceStatus{
				Code:        codes.Internal,
				Name:        "INTERNAL",
				Description: "Server error",
			},
		},
		{
			name: "UNAVAILABLE",
			code: codes.Unavailable,
			results: util.NiceStatus{
				Code:        codes.Unavailable,
				Name:        "UNAVAILABLE",
				Description: "Unavailable",
			},
		},
		{
			name: "DATA_LOSS",
			code: codes.DataLoss,
			results: util.NiceStatus{
				Code:        codes.DataLoss,
				Name:        "DATA_LOSS",
				Description: "Data loss",
			},
		},
		{
			name: "UNAUTHENTICATED",
			code: codes.Unauthenticated,
			results: util.NiceStatus{
				Code:        codes.Unauthenticated,
				Name:        "UNAUTHENTICATED",
				Description: "Unauthenticated",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := util.GetNiceStatus(tt.code)
			require.Equal(t, tt.results.Code, s.Code)
			require.Equal(t, tt.results.Name, s.Name)
			require.Equal(t, tt.results.Description, s.Description)
		})
	}
}

func TestNilNiceStatusReturnsNilGRPCStatus(t *testing.T) {
	t.Parallel()

	var ns *util.NiceStatus
	require.Nil(t, ns.GRPCStatus())
}

func TestNilNiceStatusReturnsOKError(t *testing.T) {
	t.Parallel()

	var ns *util.NiceStatus
	require.Equal(t, "OK", ns.Error())
}
