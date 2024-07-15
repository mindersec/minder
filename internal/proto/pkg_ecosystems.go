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

package proto

// AsString returns the string representation of the DepEcosystem
func (ecosystem DepEcosystem) AsString() string {
	switch ecosystem {
	case DepEcosystem_DEP_ECOSYSTEM_NPM:
		return "npm"
	case DepEcosystem_DEP_ECOSYSTEM_GO:
		return "Go"
	case DepEcosystem_DEP_ECOSYSTEM_PYPI:
		return "PyPI"
	case DepEcosystem_DEP_ECOSYSTEM_UNSPECIFIED:
		// this shouldn't happen
		return ""
	default:
		return ""
	}
}
