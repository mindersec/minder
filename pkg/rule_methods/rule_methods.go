package rule_methods

import (
	"context"
	"encoding/json"

	container "github.com/stacklok/mediator/pkg/container"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

type RuleMethods struct{}
type ValidateSignatureResult struct {
	Verification   interface{}
	GithubWorkflow interface{}
}

// ValidateSignature validates the signature of the image
func (rm RuleMethods) ValidateSignature(ctx context.Context, client ghclient.RestAPI, accessToken string,
	containerData *pb.ArtifactEventPayload) (json.RawMessage, error) {
	if containerData.ArtifactType == "CONTAINER" {
		signature_verification, github_workflow, err := container.ValidateSignature(ctx, client, accessToken, containerData.OwnerLogin,
			containerData.ArtifactName, containerData.PackageUrl)
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
