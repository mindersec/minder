// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ptr contains the Ptr function
package ptr

// Ptr takes an argument and returns a pointer to it
// this is useful when instantiating structs whose fields are pointers to basic
// types
func Ptr[T any](val T) *T {
	return &val
}

// PtrValueOrZero returns the value of a pointer or the zero value if nil
func PtrValueOrZero[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}
