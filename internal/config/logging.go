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

package config

import "github.com/spf13/viper"

// LoggingConfig is the configuration for the logging package
type LoggingConfig struct {
	Level   string `mapstructure:"level"`
	Format  string `mapstructure:"format"`
	LogFile string `mapstructure:"logFile"`
}

// SetLoggingViperDefaults sets the default values for the logging configuration
// to be picked up by viper
func SetLoggingViperDefaults(v *viper.Viper) {
	v.SetDefault("logging.level", "debug")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.logFile", "")
}
