// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestPrintProto(t *testing.T) {
	t.Parallel()

	repo := &minderv1.Repository{
		Owner: "mindersec",
		Name:  "minder",
	}

	t.Run("JSON format writes to OutOrStdout", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := cli.PrintProto(cmd, "json", repo)
		require.NoError(t, err)

		var parsed map[string]any
		err = json.Unmarshal(buf.Bytes(), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "mindersec", parsed["owner"])
		assert.Equal(t, "minder", parsed["name"])
	})

	t.Run("YAML format writes to OutOrStdout", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := cli.PrintProto(cmd, "yaml", repo)
		require.NoError(t, err)

		var parsed map[string]any
		err = yaml.Unmarshal(buf.Bytes(), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "mindersec", parsed["owner"])
		assert.Equal(t, "minder", parsed["name"])
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		t.Parallel()
		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := cli.PrintProto(cmd, "xml", repo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported output format")
	})
}

func TestPrintMultiProtoYAML(t *testing.T) {
	t.Parallel()

	repos := []*minderv1.Repository{
		{Owner: "owner1", Name: "repo1"},
		{Owner: "owner2", Name: "repo2"},
	}

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cli.PrintMultiProtoYAML(cmd, repos)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "name: repo1")
	assert.Contains(t, out, "owner: owner1")
	assert.Contains(t, out, "name: repo2")
	assert.Contains(t, out, "owner: owner2")
	assert.Contains(t, out, "---")
}
