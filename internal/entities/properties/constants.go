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

package properties

// General entity keys
const (
	// PropertyName represents the name of the entity. The name is formatted by the provider
	PropertyName = "name"
	// PropertyUpstreamID represents the ID of the entity in the provider
	PropertyUpstreamID = "upstream_id"
)

// Repository property keys
const (
	// RepoPropertyIsPrivate represents whether the repository is private
	RepoPropertyIsPrivate = "is_private"
	// RepoPropertyIsArchived represents whether the repository is archived
	RepoPropertyIsArchived = "is_archived"
	// RepoPropertyIsFork represents whether the repository is a fork
	RepoPropertyIsFork = "is_fork"
)
