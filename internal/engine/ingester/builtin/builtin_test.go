// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package builtin provides the builtin ingestion engine
// this test is directly in the builtin package because it is testing the internals of the ingestor and setting
// the rule methods to a fake
package builtin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestBuiltInWorks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		methodName string
		ent        protoreflect.ProtoMessage
		params     map[string]any
		ingested   map[string]any
	}{
		{
			name:       "passthrough works",
			methodName: "Passthrough",
			ent: &pb.Artifact{
				Name: "test",
			},
			ingested: map[string]any{
				"name": "test",
			},
			params: map[string]any{
				"name": "test",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bi, err := NewRuleDataIngest(nil)
			assert.NoError(t, err)
			bi.method = tt.methodName

			res, err := bi.Ingest(context.Background(), tt.ent, tt.params)
			assert.NoError(t, err)
			assert.Equal(t, tt.ingested, res.Object)
		})
	}
}

func TestBuiltinErrorCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		methodName string
		expErr     error
		ent        protoreflect.ProtoMessage
		params     map[string]any
	}{
		{
			name:       "entity doesn't match",
			methodName: "Passthrough",
			expErr:     evalerrors.ErrEvaluationSkipSilently,
			ent:        &pb.Artifact{},
			params: map[string]any{
				"foo": "bar",
			},
		},
		{
			name:       "method doesn't match",
			methodName: "nosuchmethod",
			expErr:     nil, // there's no specific error for this
			ent:        &pb.Artifact{},
			params: map[string]any{
				"foo": "bar",
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bi, err := NewRuleDataIngest(nil)
			assert.NoError(t, err)
			bi.method = tt.methodName

			res, err := bi.Ingest(context.Background(), tt.ent, tt.params)
			assert.Error(t, err, "expected error")
			assert.Nil(t, res)
			if tt.expErr != nil {
				assert.Equal(t, tt.expErr, err)
			}
		})
	}
}
