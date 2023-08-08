// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

// Package rule_methods provides the methods that are used by the rules
package rule_methods

import (
	"context"
	"encoding/json"

	container "github.com/stacklok/mediator/pkg/container"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// RuleMethods is the struct that contains the methods that are used by the rules
type RuleMethods struct{}

// ValidateSignatureResult is the struct that contains the result of the ValidateSignature method
type ValidateSignatureResult struct {
	Verification   interface{}
	GithubWorkflow interface{}
}

// ValidateSignature validates the signature of the image
func (_ RuleMethods) ValidateSignature(ctx context.Context, accessToken string,
	containerData *pb.ArtifactEventPayload) (json.RawMessage, error) {
	if containerData.ArtifactType == "CONTAINER" {
		signature_verification, github_workflow, err := container.ValidateSignature(ctx, accessToken, containerData.OwnerLogin,
			containerData.PackageUrl)
		if err != nil {
			return nil, err
		}
		result := ValidateSignatureResult{Verification: signature_verification, GithubWorkflow: github_workflow}
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(jsonBytes), nil
	}
	return nil, nil
}
