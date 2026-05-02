// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestGetOrderedYamlFromProto_Profile(t *testing.T) {
	t.Parallel()

	p := &pb.Profile{
		Version: "v1",
		Type:    "profile",
		Name:    "my-profile",
	}

	out, err := util.GetOrderedYamlFromProto(p)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.GreaterOrEqual(t, len(lines), 3, "expected at least 3 lines of YAML output")
	assert.True(t, strings.HasPrefix(lines[0], "version:"), "first field should be version, got: %s", lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "type:"), "second field should be type, got: %s", lines[1])
	assert.True(t, strings.HasPrefix(lines[2], "name:"), "third field should be name, got: %s", lines[2])
}

func TestGetOrderedYamlFromProto_RuleType(t *testing.T) {
	t.Parallel()

	rt := &pb.RuleType{
		Version: "v1",
		Type:    "rule-type",
		Name:    "my-rule",
	}

	out, err := util.GetOrderedYamlFromProto(rt)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.GreaterOrEqual(t, len(lines), 3, "expected at least 3 lines of YAML output")
	assert.True(t, strings.HasPrefix(lines[0], "version:"), "first field should be version, got: %s", lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "type:"), "second field should be type, got: %s", lines[1])
	assert.True(t, strings.HasPrefix(lines[2], "name:"), "third field should be name, got: %s", lines[2])
}

func TestGetOrderedYamlFromProto_DataSource(t *testing.T) {
	t.Parallel()

	ds := &pb.DataSource{
		Version: "v1",
		Type:    "data-source",
		Name:    "my-datasource",
	}

	out, err := util.GetOrderedYamlFromProto(ds)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.GreaterOrEqual(t, len(lines), 3, "expected at least 3 lines of YAML output")
	assert.True(t, strings.HasPrefix(lines[0], "version:"), "first field should be version, got: %s", lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "type:"), "second field should be type, got: %s", lines[1])
	assert.True(t, strings.HasPrefix(lines[2], "name:"), "third field should be name, got: %s", lines[2])
}

func TestGetOrderedYamlFromProto_NonResourceFallsBackToAlpha(t *testing.T) {
	t.Parallel()

	// Repository is not a resource type — should fall back to alphabetical order (name before owner)
	repo := &pb.Repository{
		Owner: "my-org",
		Name:  "my-repo",
	}

	out, err := util.GetOrderedYamlFromProto(repo)
	require.NoError(t, err)
	assert.NotEmpty(t, out)
	// alphabetical: "name" comes before "owner"
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.True(t, strings.HasPrefix(lines[0], "name:"), "non-resource type should be alphabetical, got: %s", lines[0])
}

func TestGetOrderedYamlFromProto_NilReturnsEmptyObject(t *testing.T) {
	t.Parallel()

	out, err := util.GetOrderedYamlFromProto(nil)
	require.NoError(t, err)
	assert.Equal(t, "{}\n", out)
}

func TestGetOrderedYamlFromProto_ContextComesAfterName(t *testing.T) {
	t.Parallel()

	project := "proj-123"
	p := &pb.Profile{
		Version: "v1",
		Type:    "profile",
		Name:    "test-profile",
		Context: &pb.Context{Project: &project},
	}

	out, err := util.GetOrderedYamlFromProto(p)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.True(t, strings.HasPrefix(lines[0], "version:"), "line 0: %s", lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "type:"), "line 1: %s", lines[1])
	assert.True(t, strings.HasPrefix(lines[2], "name:"), "line 2: %s", lines[2])
	assert.True(t, strings.HasPrefix(lines[3], "context:"), "line 3: %s", lines[3])
}
