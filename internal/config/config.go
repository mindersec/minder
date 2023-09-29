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
	Salt          CryptoConfig       `mapstructure:"salt"`
	Auth          AuthConfig         `mapstructure:"auth"`
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
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
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
