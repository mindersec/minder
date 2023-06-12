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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

// Package util provides helper functions for the mediator CLI.
package util

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// NiceStatus A wrapper around a status to give a better description.
type NiceStatus struct {
	// Descritpion status code
	Code codes.Code
	// Name
	Name string
	// Description
	Description string
	// Actions, reasons and links
	Details string
}

// GetNiceStatus get a nice status from the code.
func GetNiceStatus(code codes.Code) *NiceStatus {
	s := &NiceStatus{}
	return s.SetCode(code)
}

// SetCode generates the nice status from the code.
//
//nolint:gocyclo
func (s *NiceStatus) SetCode(code codes.Code) *NiceStatus {
	s.Code = code
	switch code {
	case codes.OK:
		s.Name = "OK"
		s.Description = "OK"
		s.Details = `OK is returned on success.`
	case codes.Canceled:
		s.Name = "CANCELLED"
		s.Description = "Cancelled"
		s.Details = `Canceled indicates the operation was canceled (typically by the caller).`
	case codes.Unknown:
		s.Name = "UNKNOWN"
		s.Description = "Unknown"
		s.Details = `Unknown error.`
	case codes.InvalidArgument:
		s.Name = "INVALID_ARGUMENT"
		s.Description = "Invalid argument"
		s.Details = `InvalidArgument indicates client specified an invalid argument.`
	case codes.DeadlineExceeded:
		s.Name = "DEADLINE_EXCEEDED"
		s.Description = "Deadline exceeded"
		s.Details = `DeadlineExceeded means operation expired before completion.`
	case codes.NotFound:
		s.Name = "NOT_FOUND"
		s.Description = "Not found"
		s.Details = `NotFound means some requested entity (e.g., file or directory) was
not found.`
	case codes.AlreadyExists:
		s.Name = "ALREADY_EXISTS"
		s.Description = "Already exists"
		s.Details = `AlreadyExists means an attempt to create an entity failed because one
already exists.`
	case codes.PermissionDenied:
		s.Name = "PERMISSION_DENIED"
		s.Description = "Permission denied"
		s.Details = `PermissionDenied indicates the caller does not have permission to
execute the specified operation.`
	case codes.ResourceExhausted:
		s.Name = "RESOURCE_EXHAUSTED"
		s.Description = "Resource exhausted"
		s.Details = `ResourceExhausted indicates some resource has been exhausted, perhaps
a per-user quota, or perhaps the entire file system is out of space.`
	case codes.FailedPrecondition:
		s.Name = "FAILED_PRECONDITION"
		s.Description = "Failed precondition"
		s.Details = `FailedPrecondition indicates operation was rejected because the
system is not in a state required for the operation's execution.`
	case codes.Aborted:
		s.Name = "ABORTED"
		s.Description = "Aborted"
		s.Details = `Aborted indicates the operation was aborted, typically due to a
concurrency issue like sequencer check failures, transaction aborts, etc.`
	case codes.OutOfRange:
		s.Name = "OUT_OF_RANGE"
		s.Description = "Out of range"
		s.Details = `OutOfRange means operation was attempted past the valid range.
E.g., seeking or reading past end of file.`
	case codes.Unimplemented:
		s.Name = "UNIMPLEMENTED"
		s.Description = "Unimplemented"
		s.Details = `Unimplemented indicates operation is not implemented or not
supported/enabled in this service.`
	case codes.Internal:
		s.Name = "INTERNAL"
		s.Description = "Server error"
		s.Details = `Internal errors. Means some invariants expected by underlying
system has been broken.
Please check with the server team.`
	case codes.Unavailable:
		s.Name = "UNAVAILABLE"
		s.Description = "Unavailable"
		s.Details = `The service is currently unavailable
This is a most likely a transient condition and may be corrected
by retrying with a backoff.`
	case codes.DataLoss:
		s.Name = "DATA_LOSS"
		s.Description = "Data loss"
		s.Details = `DataLoss indicates unrecoverable data loss or corruption.`
	case codes.Unauthenticated:
		s.Name = "UNAUTHENTICATED"
		s.Description = "Unauthenticated"
		s.Details = `Unauthenticated indicates the request does not have valid
authentication credentials for the operation.`
	}
	return s
}

// String convert the status to a string
func (s *NiceStatus) String() string {
	ret := fmt.Sprintf("Code: %d\nName: %s\nDescription: %s\nDetails: %s", s.Code, s.Name, s.Description, s.Details)
	return ret
}
