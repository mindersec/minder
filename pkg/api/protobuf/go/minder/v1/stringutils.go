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
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

func enumToStringViaDescriptor(d protoreflect.EnumDescriptor, n protoreflect.EnumNumber) string {
	implVal := d.Values().ByNumber(n)
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
