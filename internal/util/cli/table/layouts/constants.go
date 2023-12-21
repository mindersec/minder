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

// Package layouts defines the available table layouts
package layouts

// TableLayout is the type for table layouts
type TableLayout string

const (
	// KeyValue is the key value table layout
	KeyValue TableLayout = "keyvalue"
	// RuleType is the rule type table layout
	RuleType TableLayout = "ruletype"
	// ProfileSettings is the profile settings table layout
	ProfileSettings TableLayout = "profile_settings"
	// Profile is the profile table layout
	Profile TableLayout = "profile"
	// RepoList is the repo list table layout
	RepoList TableLayout = "repolist"
	// ProfileStatus is the profile status table layout
	ProfileStatus TableLayout = "profile_status"
	// RuleEvaluations is the rule evaluations table layout
	RuleEvaluations TableLayout = "rule_evaluations"
	// Default is the default table layout
	Default TableLayout = ""
)
