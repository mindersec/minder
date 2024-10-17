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
