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

// Package config contains a centralized structure for all configuration
// options.
package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config is the top-level configuration structure.
type Config struct {
	HTTPServer    HTTPServerConfig   `mapstructure:"http_server"`
	GRPCServer    GRPCServerConfig   `mapstructure:"grpc_server"`
	MetricServer  MetricServerConfig `mapstructure:"metric_server"`
	LoggingConfig LoggingConfig      `mapstructure:"logging"`
	Tracing       TracingConfig      `mapstructure:"tracing"`
	Metrics       MetricsConfig      `mapstructure:"metrics"`
	Database      DatabaseConfig     `mapstructure:"database"`
	Identity      IdentityConfig     `mapstructure:"identity"`
	Auth          AuthConfig         `mapstructure:"auth"`
	WebhookConfig WebhookConfig      `mapstructure:"webhook-config"`
	Events        EventConfig        `mapstructure:"events"`
}

// DefaultConfigForTest returns a configuration with all the struct defaults set,
// but no other changes.
func DefaultConfigForTest() *Config {
	v := viper.New()
	SetViperDefaults(v)
	c, err := ReadConfigFromViper(v)
	if err != nil {
		panic(fmt.Sprintf("Failed to read default config: %v", err))
	}
	return c
}

// ReadConfigFromViper reads the configuration from the given Viper instance.
// This will return the already-parsed and validated configuration, or an error.
func ReadConfigFromViper(v *viper.Viper) (*Config, error) {
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SetViperDefaults sets the default values for the configuration to be picked
// up by viper
func SetViperDefaults(v *viper.Viper) {
	v.SetEnvPrefix("minder")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setViperStructDefaults(v, "", Config{})
}

// setViperStructDefaults recursively sets the viper default values for the given struct.
//
// Per https://github.com/spf13/viper/issues/188#issuecomment-255519149, and
// https://github.com/spf13/viper/issues/761, we need to call viper.SetDefault() for each
// field in the struct to be able to use env var overrides.  This also lets us use the
// struct as the source of default values, so yay?
func setViperStructDefaults(v *viper.Viper, prefix string, s any) {
	structType := reflect.TypeOf(s)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if unicode.IsLower([]rune(field.Name)[0]) {
			// Skip private fields
			continue
		}
		if field.Tag.Get("mapstructure") == "" {
			// Error, need a tag
			panic(fmt.Sprintf("Untagged config struct field %q", field.Name))
		}
		valueName := strings.ToLower(prefix + field.Tag.Get("mapstructure"))

		if field.Type.Kind() == reflect.Struct {
			setViperStructDefaults(v, valueName+".", reflect.Zero(field.Type).Interface())
			continue
		}

		// Extract a default value the `default` struct tag
		// we don't support all value types yet, but we can add them as needed
		value := field.Tag.Get("default")
		defaultValue := reflect.Zero(field.Type).Interface()
		var err error // We handle errors at the end of the switch
		fieldType := field.Type.Kind()
		//nolint:golint,exhaustive
		switch fieldType {
		case reflect.String:
			defaultValue = value
		case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int,
			reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
			defaultValue, err = strconv.Atoi(value)
		case reflect.Float64:
			defaultValue, err = strconv.ParseFloat(value, 64)
		case reflect.Bool:
			defaultValue, err = strconv.ParseBool(value)
		default:
			err = fmt.Errorf("unhandled type %s", fieldType)
		}
		if err != nil {
			// This is effectively a compile-time error, so exit early
			panic(fmt.Sprintf("Bad value for field %q (%s): %q", valueName, fieldType, err))
		}

		if err := v.BindEnv(strings.ToUpper(valueName)); err != nil {
			panic(fmt.Sprintf("Failed to bind %q to env var: %v", valueName, err))
		}
		v.SetDefault(valueName, defaultValue)
	}
}

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
