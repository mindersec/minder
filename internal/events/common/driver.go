// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package common contains common interfaces and types used by the eventer.
package common

// DriverCloser is a function that can be used to close an eventer driver
type DriverCloser func()
