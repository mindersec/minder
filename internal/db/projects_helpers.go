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

package db

// TODO(jaosorior): Currently we have the caveat that GetChildrenProjects and GetParentProjects
// will also return the calling project. I didn't quite figure out how to
// filter this out with the CTE query. I think it's possible, but I'm not
// sure how to do it. For now, we'll just filter it out in the code.
// Once we figure out how to do it in the query, we can remove the filtering
// in the code and remove the +1 in the hierarchy offset and set it to 0.
const hierarchyOffset = 1

// CalculateProjectHierarchyOffset will calculate the offset for the hierarchy
// in the returned array from GetChildrenProjects and GetParentProjects.
// This is because the calling project is also returned.
func CalculateProjectHierarchyOffset(hierarchy int) int {
	return hierarchy + hierarchyOffset
}
