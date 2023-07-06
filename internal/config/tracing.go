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

// TracingConfig is the configuration for our tracing capabilities
type TracingConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// for the demonstration, we use AlwaysSmaple sampler to take all spans.
	// do not use this option in production.
	SampleRatio float64 `mapstructure:"sample_ratio"`
}

// SetTracingViperDefaults sets the default values for the tracing configuration
// to be picked up by viper
func SetTracingViperDefaults(v *viper.Viper) {
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.sample_ratio", 0.1)
}
