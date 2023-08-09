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

// ValidateSignature validates the signature of the image
// nolint: revive
func (rm RuleMethods) ValidateSignature(ctx context.Context, accessToken string,
	containerData *pb.ArtifactEventPayload) (json.RawMessage, error) {
	if containerData.ArtifactType == "CONTAINER" {
		signature_verification, github_workflow, err := container.ValidateSignature(ctx, accessToken, containerData.OwnerLogin,
			containerData.PackageUrl)
		if err != nil {
			return nil, err
		}
		result := struct {
			Verification   any
			GithubWorkflow any
		}{signature_verification, github_workflow}
		jsonBytes, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(jsonBytes), nil
	}
	return nil, nil
}
