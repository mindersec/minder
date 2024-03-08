// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bundles

import v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"

type Metadata struct {
	BundleName string
	Version    string
	Namespace  string
	Profiles   []string
	RuleTypes  []string
}

type Bundle interface {
	GetMetadata() Metadata
	// should accept namespaced name of profile as argument
	GetProfile(profileName string) (*v1.Profile, error)
	// should accept namespaced name of rule type as argument
	GetRuleType(ruleTypeName string) (*v1.RuleType, error)
}
