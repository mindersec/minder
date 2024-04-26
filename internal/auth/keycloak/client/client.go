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

// Package client contains an auto-generated KeyCloak client from the OpenAPI spec.
package client

//go:generate curl -L -o keycloak-api.yaml https://www.keycloak.org/docs-api/24.0.1/rest-api/openapi.yaml
//go:generate oapi-codegen --config=oapi-config.yaml keycloak-api.yaml
