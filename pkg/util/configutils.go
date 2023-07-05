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

package util

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// FlagInst is a function that creates a flag and returns a pointer to the value
type FlagInst[V any] func(name string, value V, usage string) *V

// FlagInstShort is a function that creates a flag and returns a pointer to the value
type FlagInstShort[V any] func(name, shorthand string, value V, usage string) *V

// BindConfigFlag is a helper function that binds a configuration value to a flag.
//
// Parameters:
// - v: The viper.Viper object used to retrieve the configuration value.
// - flags: The pflag.FlagSet object used to retrieve the flag value.
// - viperPath: The path used to retrieve the configuration value from Viper.
// - cmdLineArg: The flag name used to check if the flag has been set and to retrieve its value.
// - help: The help text for the flag.
// - defaultValue: A default value used to determine the type of the flag (string, int, etc.).
// - binder: A function that creates a flag and returns a pointer to the value.
func BindConfigFlag[V any](
	v *viper.Viper,
	flags *pflag.FlagSet,
	viperPath string,
	cmdLineArg string,
	defaultValue V,
	help string,
	binder FlagInst[V],
) error {
	binder(cmdLineArg, defaultValue, help)
	return doViperBind[V](v, flags, viperPath, cmdLineArg, defaultValue)
}

// BindConfigFlagWithShort is a helper function that binds a configuration value to a flag.
//
// Parameters:
// - v: The viper.Viper object used to retrieve the configuration value.
// - flags: The pflag.FlagSet object used to retrieve the flag value.
// - viperPath: The path used to retrieve the configuration value from Viper.
// - cmdLineArg: The flag name used to check if the flag has been set and to retrieve its value.
// - short: The short name for the flag.
// - help: The help text for the flag.
// - defaultValue: A default value used to determine the type of the flag (string, int, etc.).
// - binder: A function that creates a flag and returns a pointer to the value.
func BindConfigFlagWithShort[V any](
	v *viper.Viper,
	flags *pflag.FlagSet,
	viperPath string,
	cmdLineArg string,
	short string,
	defaultValue V,
	help string,
	binder FlagInstShort[V],
) error {
	binder(cmdLineArg, short, defaultValue, help)
	return doViperBind[V](v, flags, viperPath, cmdLineArg, defaultValue)
}

func doViperBind[V any](
	v *viper.Viper,
	flags *pflag.FlagSet,
	viperPath string,
	cmdLineArg string,
	defaultValue V,
) error {
	v.SetDefault(viperPath, defaultValue)
	if err := v.BindPFlag(viperPath, flags.Lookup(cmdLineArg)); err != nil {
		return fmt.Errorf("failed to bind flag %s to viper path %s: %w", cmdLineArg, viperPath, err)
	}

	return nil
}
