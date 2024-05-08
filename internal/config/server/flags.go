//
// Copyright 2024 Stacklok, Inc.
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

package server

// FlagsConfig contains the configuration for feature flags
type FlagsConfig struct {
	AppName string `mapstructure:"app_name" default:"minder"`

	GoFeature GoFeatureConfig `mapstructure:"go_feature"`
}

// GoFeatureConfig contains the configuration for the GoFeatureFlag (https://gofeatureflag.org/) provider.
type GoFeatureConfig struct {
	FilePath string `mapstructure:"file_path" default:""`
}
