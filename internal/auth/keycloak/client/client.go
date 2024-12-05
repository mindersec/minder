// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package client contains an auto-generated KeyCloak client from the OpenAPI spec.
package client

//go:generate curl -L -o keycloak-api.yaml https://www.keycloak.org/docs-api/25.0.6/rest-api/openapi.yaml
//go:generate oapi-codegen --config=oapi-config.yaml keycloak-api.yaml
