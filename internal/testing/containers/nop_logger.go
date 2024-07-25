//
// Copyright 2024 Stacklok, Inc.
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

package containers

import (
	tc "github.com/testcontainers/testcontainers-go"
)

type nopLogger struct{}

func (_ *nopLogger) Printf(_ string, _ ...interface{}) {}

var _ tc.Logging = (*nopLogger)(nil)

type nopLoggerCustomizer struct{}

func (_ *nopLoggerCustomizer) Customize(req *tc.GenericContainerRequest) error {
	req.Logger = &nopLogger{}
	return nil
}

// WithNopLogger customizes container configuration inhibiting log
// messages.
func WithNopLogger() tc.ContainerCustomizer {
	return &nopLoggerCustomizer{}
}
