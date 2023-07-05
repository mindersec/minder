// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util_test

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/mediator/pkg/util"
)

const (
	viperPath  = "test.path"
	cmdLineArg = "test-arg"
	help       = "test help"
)

func TestBindConfigFlagStringWithArg(t *testing.T) {
	t.Parallel()

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	defaultValue := "test"

	err := util.BindConfigFlag(
		v, flags, viperPath, cmdLineArg, defaultValue,
		help, flags.String)

	require.NoError(t, err, "Unexpected error")

	// Check that the flags are registered
	require.NoError(t, flags.Parse([]string{"--" + cmdLineArg + "=foo"}))
	require.Equal(t, "foo", v.GetString(viperPath))
}

func TestBindConfigFlagStringWithDefaultArg(t *testing.T) {
	t.Parallel()

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	defaultValue := "test"

	err := util.BindConfigFlag(
		v, flags, viperPath, cmdLineArg, defaultValue,
		help, flags.String)

	require.NoError(t, err, "Unexpected error")

	// Check that the flags are registered
	require.NoError(t, flags.Parse([]string{}))
	require.Equal(t, defaultValue, v.GetString(viperPath))
}

func TestBindConfigFlagIntWithArg(t *testing.T) {
	t.Parallel()

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	defaultValue := 123

	err := util.BindConfigFlag(
		v, flags, viperPath, cmdLineArg, defaultValue,
		help, flags.Int)

	require.NoError(t, err, "Unexpected error")

	// Check that the flags are registered
	require.NoError(t, flags.Parse([]string{"--" + cmdLineArg + "=456"}))
	require.Equal(t, 456, v.GetInt(viperPath))
}

func TestBindConfigFlagIntWithDefaultArg(t *testing.T) {
	t.Parallel()

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	defaultValue := 123

	err := util.BindConfigFlag(
		v, flags, viperPath, cmdLineArg, defaultValue,
		help, flags.Int)

	require.NoError(t, err, "Unexpected error")

	// Check that the flags are registered
	require.NoError(t, flags.Parse([]string{}))
	require.Equal(t, 123, v.GetInt(viperPath))
}
