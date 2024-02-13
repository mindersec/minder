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

			// test nil error
			st := status.Convert(err)
			nicest := util.FromRpcError(st)
			require.Equal(t, codes.OK, nicest.Code)
			require.Equal(t, "OK", nicest.Name)
		})
	}
}
