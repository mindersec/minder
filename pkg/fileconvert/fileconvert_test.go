// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package fileconvert

import (
	"bytes"
	"cmp"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestReadResource(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    proto.Message
		wantErr bool
	}{
		{
			name: "valid profile",
			input: `
type: profile
version: v1
name: test-profile
repository:
  - type: sample-rule
    def: {}
`,
			want: &minderv1.Profile{
				Type:    string(minderv1.ProfileResource),
				Version: "v1",
				Name:    "test-profile",
				Repository: []*minderv1.Profile_Rule{
					{
						Type: "sample-rule",
						Def:  &structpb.Struct{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid rule type",
			input: `
type: rule-type
version: v1
name: test-rule-type
def:
  in_entity: "artifact"
  rule_schema: {}
  ingest:
    type: fake
  eval:
    type: other
`,
			want: &minderv1.RuleType{
				Type:    string(minderv1.RuleTypeResource),
				Version: "v1",
				Name:    "test-rule-type",
				Def: &minderv1.RuleType_Definition{
					InEntity:   "artifact",
					RuleSchema: &structpb.Struct{},
					Ingest: &minderv1.RuleType_Definition_Ingest{
						Type: "fake",
					},
					Eval: &minderv1.RuleType_Definition_Eval{
						Type: "other",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid data source",
			input: `
type: data-source
version: v1
name: test-data-source
rest:
  def:
    function:
      endpoint: http://example.com/
      input_schema: {}
`,
			want: &minderv1.DataSource{
				Type:    string(minderv1.DataSourceResource),
				Version: "v1",
				Name:    "test-data-source",
				Driver: &minderv1.DataSource_Rest{
					Rest: &minderv1.RestDataSource{
						Def: map[string]*minderv1.RestDataSource_Def{
							"function": {
								Endpoint:    "http://example.com/",
								InputSchema: &structpb.Struct{},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "rule type validate fails",
			input: `
type: rule-type
version: v1
name: test-rule-type
# def is required
`,
			wantErr: true,
		},
		{
			name: "invalid version",
			input: `
type: profile
version: v2
`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing type",
			input: `
version: v1
`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "unknown resource type",
			input: `
type: unknown
version: v1
`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			decoder := yaml.NewDecoder(bytes.NewBufferString(tt.input))
			got, err := ReadResource(decoder)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if diff := gocmp.Diff(got, tt.want, protocmp.Transform()); diff != "" {
				t.Errorf("ReadResource on \n%s\n\n%s", tt.input, diff)
			}
		})
	}
}

func TestReadResourceTyped(t *testing.T) {
	t.Parallel()

	profileDecoder, profileCloser := DecoderForFile("testdata/directory/profile.json")
	require.NotNil(t, profileDecoder, "Expected non-nil decoder for profile")
	t.Cleanup(func() { _ = profileCloser.Close() })
	_, err := ReadResourceTyped[*minderv1.Profile](profileDecoder)
	require.NoError(t, err, "Expected no error reading profile")

	dataSourceDecoder, dataSourceCloser := DecoderForFile("testdata/directory/datasource.yaml")
	require.NotNil(t, dataSourceDecoder, "Expected non-nil decoder for profile")
	t.Cleanup(func() { _ = dataSourceCloser.Close() })
	_, err = ReadResourceTyped[*minderv1.DataSource](dataSourceDecoder)
	require.NoError(t, err, "Expected no error reading profile")

	ruleTypeDecoder, ruleTypeCloser := DecoderForFile("testdata/directory/ruletype.yaml")
	require.NotNil(t, ruleTypeDecoder, "Expected non-nil decoder for profile")
	t.Cleanup(func() { _ = ruleTypeCloser.Close() })
	_, err = ReadResourceTyped[*minderv1.RuleType](ruleTypeDecoder)
	require.NoError(t, err, "Expected no error reading profile")
}

func TestReadAll(t *testing.T) {
	t.Parallel()

	dirResources, err := ResourcesFromPaths(t.Logf, "testdata/directory")
	require.NoError(t, err)

	collectedInput, closer := DecoderForFile("testdata/resources.yaml")
	require.NotNil(t, collectedInput, "Expected non-nil decoder for profile")
	t.Cleanup(func() { _ = closer.Close() })
	collected := make([]proto.Message, 0, 3)
	for {
		resource, err := ReadResource(collectedInput)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		collected = append(collected, resource)
	}

	assert.Equal(t, len(collected), len(dirResources))
	for i, dirResource := range dirResources {
		diff := gocmp.Diff(dirResource, collected[i], protocmp.Transform())
		if diff != "" {
			t.Errorf("Read resources did not match expected (-dir,+file):\n%s", diff)
		}
	}
}

func TestReadWriteRoundTrip(t *testing.T) {
	t.Parallel()

	dirResources, err := ResourcesFromPaths(t.Logf, "testdata/directory")
	require.NoError(t, err)

	tempFile := filepath.Clean(filepath.Join(t.TempDir(), "collected.yaml"))
	outFile, err := os.Create(tempFile)
	require.NoError(t, err)
	t.Cleanup(func() { _ = outFile.Close() })

	slices.SortFunc(dirResources, func(a, b minderv1.ResourceMeta) int {
		return cmp.Or(
			strings.Compare(a.GetType(), b.GetType()),
			strings.Compare(a.GetName(), b.GetName()),
		)
	})

	encoder := yaml.NewEncoder(outFile)
	encoder.SetIndent(2)
	for _, resource := range dirResources {
		err = WriteResource(encoder, resource)
		require.NoError(t, err)
	}

	expectedContents, err := os.ReadFile("testdata/resources.yaml")
	require.NoError(t, err)
	gotContents, err := os.ReadFile(tempFile)
	require.NoError(t, err)
	if diff := gocmp.Diff(string(expectedContents), string(gotContents)); diff != "" {
		t.Errorf("Read resources did not match expected (-want,+got):\n%s", diff)
	}
}
