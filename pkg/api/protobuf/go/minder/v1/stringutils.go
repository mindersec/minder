// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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

// enumFromStringViaDescriptor looks up an enum value by its (name) option,
// the inverse of enumToStringViaDescriptor. The second return value is false
// if no enum value has a (name) option matching s.
func enumFromStringViaDescriptor(d protoreflect.EnumDescriptor, s string) (protoreflect.EnumNumber, bool) {
	values := d.Values()
	for i := 0; i < values.Len(); i++ {
		val := values.Get(i)
		extension := proto.GetExtension(val.Options(), E_Name)
		implName, ok := extension.(string)
		if ok && implName == s {
			return val.Number(), true
		}
	}

	return 0, false
}
