// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package fileconvert

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
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

func TestReadResourceSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
		wantMsg string
	}{
		{
			name:    "missing type",
			input:   "version: v1\n",
			wantErr: ErrResourceTypeNotFound,
			wantMsg: "resource type not found",
		},
		{
			name:    "unknown type",
			input:   "type: unknown\nversion: v1\n",
			wantErr: ErrUnknownResourceType,
			wantMsg: "unknown resource type: unknown",
		},
		{
			name:    "non-Minder type without version",
			input:   "type: unknown\nname: something\n",
			wantErr: ErrUnknownResourceType,
			wantMsg: "unknown resource type: unknown",
		},
		{
			name: "unexpected type",
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
			wantErr: ErrUnexpectedResourceType,
			wantMsg: "unexpected resource type: *v1.DataSource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			decoder := yaml.NewDecoder(bytes.NewBufferString(tt.input))
			_, err := ReadResourceTyped[*minderv1.RuleType](decoder)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.EqualError(t, err, tt.wantMsg)
		})
	}
}

// TestKnownResourceTypesInSync guards that knownResourceTypes matches the switch in ReadResource.
func TestKnownResourceTypesInSync(t *testing.T) {
	t.Parallel()

	for _, knownType := range knownResourceTypes {
		t.Run(knownType, func(t *testing.T) {
			t.Parallel()

			gotType, err := resourceType(map[string]any{"type": knownType, "version": "v1"})
			require.NoError(t, err)
			assert.Equal(t, knownType, gotType)

			decoder := yaml.NewDecoder(bytes.NewBufferString(fmt.Sprintf("type: %s\nversion: v1\n", knownType)))
			_, err = ReadResource(decoder)
			assert.NotErrorIs(t, err, ErrUnknownResourceType)
		})
	}
}

func TestReadAll(t *testing.T) {
	t.Parallel()

	dirResources, err := ResourcesFromPaths(t.Logf, "testdata/directory")
	require.NoError(t, err)

	collectedInput, closer := DecoderForFile("testdata/resources.yaml")
	require.NotNil(t, collectedInput, "Expected non-nil decoder for profile")
	t.Cleanup(func() { _ = closer.Close() })
	collected := make([]proto.Message, 0, 4)
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

func TestReadRuleTypeEnumNames(t *testing.T) {
	t.Parallel()

	const ruleTypeTmpl = `
type: rule-type
version: v1
name: test-rule-type
def:
  in_entity: repository
  rule_schema: {}
  ingest:
    type: git
  eval:
    type: other
`
	tests := []struct {
		name         string
		enums        string
		wantPhase    minderv1.RuleTypeReleasePhase
		wantSeverity minderv1.Severity_Value
		wantErr      bool
	}{
		{
			name: "short names",
			enums: `
release_phase: beta
severity:
  value: info
`,
			wantPhase:    minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_BETA,
			wantSeverity: minderv1.Severity_VALUE_INFO,
		},
		{
			name: "full proto names",
			enums: `
release_phase: RULE_TYPE_RELEASE_PHASE_BETA
severity:
  value: VALUE_INFO
`,
			wantPhase:    minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_BETA,
			wantSeverity: minderv1.Severity_VALUE_INFO,
		},
		{
			name: "json field name",
			enums: `
releasePhase: alpha
severity:
  value: high
`,
			wantPhase:    minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_ALPHA,
			wantSeverity: minderv1.Severity_VALUE_HIGH,
		},
		{
			name: "unknown release phase",
			enums: `
release_phase: gamma
`,
			wantErr: true,
		},
		{
			name: "unknown severity",
			enums: `
severity:
  value: catastrophic
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			decoder := yaml.NewDecoder(bytes.NewBufferString(ruleTypeTmpl + tt.enums))
			got, err := ReadResourceTyped[*minderv1.RuleType](decoder)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPhase, got.GetReleasePhase())
			assert.Equal(t, tt.wantSeverity, got.GetSeverity().GetValue())
		})
	}
}

// TestReadRuleTypeRego reads .rego rule types through ReadResource. Some
// METADATA keys are not RuleType fields ("title", which is copied into
// display_name, and the "custom" block, whose entries are copied to the top
// level). These keys are deleted by the decoder, which lets unmarshalRuleType
// reject unknown fields.
func TestReadRuleTypeRego(t *testing.T) {
	t.Parallel()

	// A minimal rule derived from testdata/directory/ruletype.rego, using the
	// top-level form so that "title" is the only leftover key.
	const titleOnly = `# METADATA
#
# title: Title only
# description: Only the title key is left over after decoding
# def:
#   in_entity: repository
#   ingest:
#     type: git
package minder

import rego.v1

default allow := true
`

	const opaAnnotations = `# METADATA
# scope: package
# title: OPA annotations
# authors:
# - Jane Doe <jane@example.com>
# related_resources:
# - https://example.com/policy
# def:
#   in_entity: repository
#   ingest:
#     type: git
package minder

import rego.v1

default allow := true
`

	tests := []struct {
		name            string
		decoder         func(t *testing.T) Decoder
		wantName        string
		wantDisplayName string
		wantPhase       minderv1.RuleTypeReleasePhase
		wantRegoType    string
	}{
		{
			name: "fixture with title and custom block",
			decoder: func(t *testing.T) Decoder {
				t.Helper()
				decoder, closer := DecoderForFile("testdata/directory/ruletype.rego")
				require.NotNil(t, decoder, "Expected non-nil decoder for rego rule type")
				t.Cleanup(func() { _ = closer.Close() })
				return decoder
			},
			wantName:        "ruletype",
			wantDisplayName: "Test ruletype in Rego format",
			wantPhase:       minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_ALPHA,
			wantRegoType:    "constraints",
		},
		{
			name: "title only",
			decoder: func(_ *testing.T) Decoder {
				return &regoDecoder{filename: "title_only.rego", file: strings.NewReader(titleOnly)}
			},
			wantName:        "title_only",
			wantDisplayName: "Title only",
			wantPhase:       minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED,
			wantRegoType:    "deny-by-default",
		},
		{
			name: "standard OPA annotations",
			decoder: func(_ *testing.T) Decoder {
				return &regoDecoder{filename: "opa_annotations.rego", file: strings.NewReader(opaAnnotations)}
			},
			wantName:        "opa_annotations",
			wantDisplayName: "OPA annotations",
			wantPhase:       minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_UNSPECIFIED,
			wantRegoType:    "deny-by-default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ReadResourceTyped[*minderv1.RuleType](tt.decoder(t))
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, got.GetName())
			assert.Equal(t, tt.wantDisplayName, got.GetDisplayName())
			assert.Equal(t, tt.wantPhase, got.GetReleasePhase())
			assert.Equal(t, "rego", got.GetDef().GetEval().GetType())
			assert.Equal(t, tt.wantRegoType, got.GetDef().GetEval().GetRego().GetType())
			assert.Contains(t, got.GetDef().GetEval().GetRego().GetDef(), "package minder")
		})
	}
}

func TestReadRuleTypeUnknownFields(t *testing.T) {
	t.Parallel()

	const yamlTmpl = `
type: rule-type
version: v1
name: test-rule-type
%s
def:
  in_entity: repository
  rule_schema: {}
  %s: [git]
  ingest:
    type: git
  eval:
    type: other
`
	const regoTmpl = `# METADATA
#
# title: Unknown fields
# custom:
#   %s: alpha
#   def:
#     in_entity: repository
#     ingest:
#       type: git
package minder

import rego.v1

default allow := true
`

	tests := []struct {
		name    string
		decoder Decoder
		wantErr string
	}{
		{
			name:    "yaml correct spelling",
			decoder: yaml.NewDecoder(strings.NewReader(fmt.Sprintf(yamlTmpl, "", "provider_traits"))),
		},
		{
			name:    "yaml misspelled def field",
			decoder: yaml.NewDecoder(strings.NewReader(fmt.Sprintf(yamlTmpl, "", "provider_traitss"))),
			wantErr: "provider_traitss",
		},
		{
			name: "yaml misspelled top-level field",
			decoder: yaml.NewDecoder(strings.NewReader(
				fmt.Sprintf(yamlTmpl, "short_failure_mesage: oops", "provider_traits"))),
			wantErr: "short_failure_mesage",
		},
		{
			name:    "rego correct custom key",
			decoder: &regoDecoder{filename: "ok.rego", file: strings.NewReader(fmt.Sprintf(regoTmpl, "release_phase"))},
		},
		{
			name:    "rego misspelled custom key",
			decoder: &regoDecoder{filename: "bad.rego", file: strings.NewReader(fmt.Sprintf(regoTmpl, "release_phse"))},
			wantErr: "release_phse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ReadResourceTyped[*minderv1.RuleType](tt.decoder)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestRuleTypeEnumRoundTrip(t *testing.T) {
	t.Parallel()

	ruleType := &minderv1.RuleType{
		Name: "test-rule-type",
		Def: &minderv1.RuleType_Definition{
			InEntity:   "repository",
			RuleSchema: &structpb.Struct{},
			Ingest:     &minderv1.RuleType_Definition_Ingest{Type: "git"},
			Eval:       &minderv1.RuleType_Definition_Eval{Type: "other"},
		},
		Severity:     &minderv1.Severity{Value: minderv1.Severity_VALUE_CRITICAL},
		ReleasePhase: minderv1.RuleTypeReleasePhase_RULE_TYPE_RELEASE_PHASE_DEPRECATED,
	}

	var buf bytes.Buffer
	require.NoError(t, WriteResource(yaml.NewEncoder(&buf), ruleType))

	written := buf.String()
	assert.Contains(t, written, "release_phase: deprecated")
	assert.Contains(t, written, "value: critical")

	got, err := ReadResourceTyped[*minderv1.RuleType](yaml.NewDecoder(&buf))
	require.NoError(t, err)
	if diff := gocmp.Diff(ruleType, got, protocmp.Transform()); diff != "" {
		t.Errorf("round trip mismatch (-want,+got):\n%s", diff)
	}
}
