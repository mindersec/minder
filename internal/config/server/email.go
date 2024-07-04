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

// EmailConfig is the configuration for the email sending service
type EmailConfig struct {
	// MinderURLBase is the base URL to use for minder invite URLs, e.g. https://cloud.stacklok.com
	MinderURLBase string `mapstructure:"minder_url_base"`
	// AWSSES is the AWS SES configuration
	AWSSES AWSSES `mapstructure:"aws_ses"`
}

// AWSSES is the AWS SES configuration
type AWSSES struct {
	// Sender is the email address of the sender
	Sender string `mapstructure:"sender"`
	// Region is the AWS region to use for AWS SES
	Region string `mapstructure:"region"`
}
