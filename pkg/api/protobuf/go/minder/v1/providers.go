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

package v1

import (
	"google.golang.org/protobuf/proto"
)

// ToString returns the string representation of the ProviderType
func (provt ProviderType) ToString() string {
	implVal := provt.Descriptor().Values().ByNumber(provt.Number())
	if implVal == nil {
		return ""
	}
	extension := proto.GetExtension(implVal.Options(), E_Name)
	implName, ok := extension.(string)
	if !ok {
		return ""
	}

	return implName
}
