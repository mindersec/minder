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

package github

import (
	"errors"
	"fmt"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// GetEntityName implements the Provider interface
func (c *GitHub) GetEntityName(entType minderv1.Entity, props *properties.Properties) (string, error) {
	if props == nil {
		return "", errors.New("properties are nil")
	}
	if c.propertyFetchers == nil {
		return "", errors.New("property fetchers not initialized")
	}
	fetcher := c.propertyFetchers.EntityPropertyFetcher(entType)
	if fetcher == nil {
		return "", fmt.Errorf("no fetcher found for entity type %s", entType)
	}
	return fetcher.GetName(props)
}
