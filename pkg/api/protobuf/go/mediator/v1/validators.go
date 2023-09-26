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

package _go

import (
	"fmt"
)

// Validator is an interface which allows for the validation of a struct.
type Validator interface {
	Validate() error
}

// ensure GitHubProviderConfig implements the Validator interface
var _ Validator = (*GitHubProviderConfig)(nil)

// Validate is a utility function which allows for the validation of a struct.
func (_ *GitHubProviderConfig) Validate() error {
	// Unfortunately, we don't currently have a way to add custom tags to
	// protobuf-generated structs, so we have to do this manually.
	return nil
}

// ensure RESTProviderConfig implements the Validator interface
var _ Validator = (*RESTProviderConfig)(nil)

// Validate is a utility function which allows for the validation of a struct.
func (rpcfg *RESTProviderConfig) Validate() error {
	// Unfortunately, we don't currently have a way to add custom tags to
	// protobuf-generated structs, so we have to do this manually.
	if rpcfg.GetBaseUrl() == "" {
		return fmt.Errorf("base_url is required")
	}

	return nil
}
