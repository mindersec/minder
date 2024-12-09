// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package structured

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJsonDecoder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		data    []byte
		mustErr bool
		expect  any
	}{
		{name: "normal", data: []byte(`{"a":1, "b":"abc"}`), mustErr: false},
		{name: "invalid_json", data: []byte(`a 1`), mustErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			_, err := b.Write(tc.data)
			require.NoError(t, err)

			dec := jsonDecoder{}
			res, err := dec.Parse(&b)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
		})
		dec := jsonDecoder{}
		_, err := dec.Parse(nil)
		require.Error(t, err)

	}
}

func TestYamlDecoder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		data    []byte
		mustErr bool
		expect  any
	}{
		{name: "normal", data: []byte("---\na: 1\nb:\n  - \"Hey\"\n  - \"Bye\"\n"), mustErr: false},
		{name: "invalid_yaml", data: []byte("  a 1\na: 2\n"), mustErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			_, err := b.Write(tc.data)
			require.NoError(t, err)

			dec := yamlDecoder{}
			res, err := dec.Parse(&b)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
		})
		dec := yamlDecoder{}
		_, err := dec.Parse(nil)
		require.Error(t, err)

	}
}

func TestTomlDecoder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		data    []byte
		mustErr bool
		expect  any
	}{
		{name: "normal", data: []byte("title = \"TOML Example\"\n\n[owner]\nname = \"Tom Preston-Werner\""), mustErr: false},
		{name: "invalid_toml", data: []byte("  a 1\na: 2\n"), mustErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			_, err := b.Write(tc.data)
			require.NoError(t, err)

			dec := tomlDecoder{}
			res, err := dec.Parse(&b)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)
		})
		dec := tomlDecoder{}
		_, err := dec.Parse(nil)
		require.Error(t, err)

	}
}
