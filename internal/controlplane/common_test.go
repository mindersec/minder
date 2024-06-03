// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/ptr"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestGetRemediationURLFromMetadata(t *testing.T) {
	validData := []byte(`{"pr_number": 18}`)
	t.Parallel()
	for _, tc := range []struct {
		name        string
		data        []byte
		repo        string
		expectedURL string
		mustErr     bool
	}{
		{"normal", validData, "My-Example_1.0/Test_2", "https://github.com/My-Example_1.0/Test_2/pull/18", false},
		{"invalid-slug", validData, "example", "", true},
		{"no-pr", []byte(`{}`), "example/test", "", false},
		{"no-slug", validData, "", "", true},
		{"no-slug-no-pr", []byte(`{}`), "", "", true},
		{"invalid-json", []byte(`Yo!`), "", "", true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			url, err := getRemediationURLFromMetadata(tc.data, tc.repo)
			if tc.mustErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedURL, url)
		})
	}

}

func TestGetAlertURLFromMetadata(t *testing.T) {
	t.Parallel()
	validPayload := []byte(`{"ghsa_id": "GHAS-advisory_ID_here"}`)
	for _, tc := range []struct {
		name     string
		data     []byte
		repo     string
		expected string
		mustErr  bool
	}{
		{"normal", validPayload, "example/test", "https://github.com/example/test/security/advisories/GHAS-advisory_ID_here", false},
		{"no-repo", validPayload, "", "", true},
		{"bad-json", []byte(`invalid _`), "", "", true},
		{"no-advisory", []byte(`{"ghsa_id": ""}`), "", "", true},
		{"invalid-slug", []byte(`{"ghsa_id": "abc"}`), "invalid slug", "", true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := getAlertURLFromMetadata(tc.data, tc.repo)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.Equal(t, tc.expected, res)
		})
	}
}

func Test_getProjectFromContextV2(t *testing.T) {
	t.Parallel()

	proj1 := uuid.New()
	proj2 := uuid.New()

	type args struct {
		accessor HasProtoContextV2Compat
	}
	tests := []struct {
		name    string
		args    args
		want    uuid.UUID
		wantErr bool
	}{
		{
			name: "no project",
			args: args{
				accessor: newMockHasProtoContextV2(),
			},
			want:    uuid.Nil,
			wantErr: true,
		},
		{
			name: "v1 project",
			args: args{
				accessor: newMockHasProtoContextV2().withV1(&pb.Context{
					Project: ptr.Ptr(proj1.String()),
				}),
			},
			want:    proj1,
			wantErr: false,
		},
		{
			name: "v2 project",
			args: args{
				accessor: newMockHasProtoContextV2().withV2(&pb.ContextV2{
					ProjectId: proj1.String(),
				}),
			},
			want:    proj1,
			wantErr: false,
		},
		{
			name: "v2 project wins",
			args: args{
				accessor: newMockHasProtoContextV2().withV1(&pb.Context{
					Project: ptr.Ptr(proj1.String()),
				}).withV2(&pb.ContextV2{
					ProjectId: proj2.String(),
				}),
			},
			want:    proj2,
			wantErr: false,
		},
		{
			name: "v2 project wins with malformed v1",
			args: args{
				accessor: newMockHasProtoContextV2().withV1(&pb.Context{
					Project: ptr.Ptr("malformed"),
				}).withV2(&pb.ContextV2{
					ProjectId: proj2.String(),
				}),
			},
			want:    proj2,
			wantErr: false,
		},
		{
			name: "malformed v2 project",
			args: args{
				accessor: newMockHasProtoContextV2().withV2(&pb.ContextV2{
					ProjectId: "malformed",
				}),
			},
			want:    uuid.Nil,
			wantErr: true,
		},
		{
			name: "malformed v2 project is still an error",
			args: args{
				accessor: newMockHasProtoContextV2().withV1(&pb.Context{
					Project: ptr.Ptr(proj1.String()),
				}).withV2(&pb.ContextV2{
					ProjectId: "malformed",
				}),
			},
			want:    uuid.Nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := getProjectFromContextV2Compat(tt.args.accessor)
			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

type mockHasProtoContextV2 struct {
	getV1 func() *pb.Context
	getV2 func() *pb.ContextV2
}

func emptyv1() *pb.Context {
	return nil
}

func emptyv2() *pb.ContextV2 {
	return nil
}

func newMockHasProtoContextV2() *mockHasProtoContextV2 {
	return &mockHasProtoContextV2{
		getV1: emptyv1,
		getV2: emptyv2,
	}
}

func (m *mockHasProtoContextV2) withV1(v1 *pb.Context) *mockHasProtoContextV2 {
	m.getV1 = func() *pb.Context {
		return v1
	}
	return m
}

func (m *mockHasProtoContextV2) withV2(v2 *pb.ContextV2) *mockHasProtoContextV2 {
	m.getV2 = func() *pb.ContextV2 {
		return v2
	}
	return m
}

func (m *mockHasProtoContextV2) GetContext() *pb.Context {
	return m.getV1()
}

func (m *mockHasProtoContextV2) GetContextV2() *pb.ContextV2 {
	return m.getV2()
}
