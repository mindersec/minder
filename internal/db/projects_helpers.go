// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
