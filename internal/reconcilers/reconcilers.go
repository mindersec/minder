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

// Package reconcilers provides implementations of all the reconcilers
package reconcilers

// Differences is a struct that holds the differences between two objects
type Differences struct {
	Field         string      `json:"field"`
	ActualValue   interface{} `json:"actual_value"`
	ExpectedValue interface{} `json:"expected_value"`
}
