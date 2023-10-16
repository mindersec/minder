// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

// Package security_advisory provides necessary interfaces and implementations for
// creating alerts of type security advisory.
package security_advisory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v53/github"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	enginerr "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
	provifv1 "github.com/stacklok/mediator/pkg/providers/v1"
)

const (
	// AlertType is the type of the security advisory alert engine
	AlertType = "security_advisory"
)

// Alert is the structure backing the security-advisory alert action
type Alert struct {
	actionType interfaces.ActionType
	cli        provifv1.GitHub
}

type paramsSA struct {
	Owner           string
	Repo            string
	Severity        string
	ID              string
	Summary         string
	Description     string
	Vulnerabilities []*github.AdvisoryVulnerability
	Metadata        *alertMetadata
}

type alertMetadata struct {
	ID string `json:"ghsa_id,omitempty"`
}

// NewSecurityAdvisoryAlert creates a new security-advisory alert action
func NewSecurityAdvisoryAlert(
	actionType interfaces.ActionType,
	saCfg *pb.RuleType_Definition_Alert_AlertTypeSA,
	pbuild *providers.ProviderBuilder,
) (*Alert, error) {
	if actionType == "" {
		return nil, fmt.Errorf("action type cannot be empty")
	}
	cli, err := pbuild.GetGitHub(context.Background())
	if err != nil {
		return nil, fmt.Errorf("cannot get http client: %w", err)
	}
	_ = saCfg
	return &Alert{
		actionType: actionType,
		cli:        cli,
	}, nil
}

// ParentType returns the action type of the security-advisory engine
func (alert *Alert) ParentType() interfaces.ActionType {
	return alert.actionType
}

// SubType returns the action subtype of the remediation engine
func (_ *Alert) SubType() string {
	return AlertType
}

// GetOnOffState returns the alert action state read from the profile
func (_ *Alert) GetOnOffState(p *pb.Profile) interfaces.ActionOpt {
	return interfaces.ActionOptFromString(p.Alert)
}

// Do alerts through security advisory
func (alert *Alert) Do(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	setting interfaces.ActionOpt,
	entity protoreflect.ProtoMessage,
	ruleDef map[string]any,
	ruleParams map[string]any,
	metadata *json.RawMessage,
) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("alert_type", AlertType).
		Str("cmd", string(cmd)).
		Msg("begin processing")

	// Get the parameters for the security advisory - owner, repo, etc.
	params, err := alert.getParamsForSecurityAdvisory(ctx, entity, ruleDef, ruleParams, metadata)
	if err != nil {
		return nil, fmt.Errorf("error extracting details: %w", err)
	}

	// Process the command based on the action setting
	switch setting {
	case interfaces.ActionOptOn:
		return alert.run(ctx, params, cmd)
	case interfaces.ActionOptDryRun:
		return nil, alert.runDry(ctx, params, cmd)
	case interfaces.ActionOptOff, interfaces.ActionOptUnknown:
		return nil, fmt.Errorf("unexpected action setting: %w", enginerr.ErrActionFailed)
	}
	return nil, enginerr.ErrActionSkipped
}

// run runs the security advisory action
func (alert *Alert) run(ctx context.Context, params *paramsSA, cmd interfaces.ActionCmd) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx)

	// Process the command
	switch cmd {
	// Open a security advisory
	case interfaces.ActionCmdOn:
		id, err := alert.cli.CreateSecurityAdvisory(ctx,
			params.Owner,
			params.Repo,
			params.Severity,
			params.Summary,
			params.Description,
			params.Vulnerabilities)
		if err != nil {
			return nil, fmt.Errorf("error creating security advisory: %w, %w", err, enginerr.ErrActionFailed)
		}
		newMeta, err := json.Marshal(alertMetadata{ID: id})
		if err != nil {
			return nil, fmt.Errorf("error marshalling alert metadata json: %w", err)
		}
		// Success - return the new metadata for storing the ghsa_id
		logger.Info().Str("ghsa_id", id).Msg("opened a security advisory")
		return newMeta, nil
	// Close a security advisory
	case interfaces.ActionCmdOff:
		if params.Metadata == nil || params.Metadata.ID == "" {
			return nil, fmt.Errorf("cannot close security-advisory without an ID: %w", enginerr.ErrActionSkipped)
		}
		err := alert.cli.CloseSecurityAdvisory(ctx, params.Owner, params.Repo, params.Metadata.ID)
		if err != nil {
			return nil, fmt.Errorf("error closing security advisory: %w, %w", err, enginerr.ErrActionFailed)
		}
		logger.Info().Str("ghsa_id", params.ID).Msg("closed security advisory")
		// Success - return ErrActionTurnedOff to indicate the action was successful
		return nil, fmt.Errorf("%s : %w", alert.ParentType(), enginerr.ErrActionTurnedOff)
	case interfaces.ActionCmdDoNothing:
		return nil, enginerr.ErrActionSkipped
	}
	return nil, enginerr.ErrActionSkipped
}

// runDry runs the security advisory action in dry run mode
func (alert *Alert) runDry(ctx context.Context, params *paramsSA, cmd interfaces.ActionCmd) error {
	logger := zerolog.Ctx(ctx)

	// Process the command
	switch cmd {
	// Open a security advisory
	case interfaces.ActionCmdOn:
		endpoint := fmt.Sprintf("repos/%v/%v/security-advisories", params.Owner, params.Repo)
		body := ""
		curlCmd, err := util.GenerateCurlCommand("POST", alert.cli.GetBaseURL(), endpoint, body)
		if err != nil {
			return fmt.Errorf("cannot generate curl command: %w", err)
		}
		logger.Info().Msgf("run the following curl command to open a security-advisory: \n%s\n", curlCmd)
		return nil
	// Close a security advisory
	case interfaces.ActionCmdOff:
		if params.Metadata == nil || params.Metadata.ID == "" {
			return fmt.Errorf("cannot close a security-advisory without an ID: %w", enginerr.ErrActionSkipped)
		}
		endpoint := fmt.Sprintf("repos/%v/%v/security-advisories/%v", params.Owner, params.Repo, params.ID)
		body := "{\"state\": \"closed\"}"
		curlCmd, err := util.GenerateCurlCommand("PATCH", alert.cli.GetBaseURL(), endpoint, body)
		if err != nil {
			return fmt.Errorf("cannot generate curl command to close a security-adivsory: %w", err)
		}
		logger.Info().Msgf("run the following curl command: \n%s\n", curlCmd)
	case interfaces.ActionCmdDoNothing:
		return enginerr.ErrActionSkipped
	}
	return enginerr.ErrActionSkipped
}

// getParamsForSecurityAdvisory extracts the details from the entity
func (_ *Alert) getParamsForSecurityAdvisory(
	ctx context.Context,
	entity protoreflect.ProtoMessage,
	_ map[string]any,
	_ map[string]any,
	metadata *json.RawMessage,
) (*paramsSA, error) {
	logger := zerolog.Ctx(ctx)
	params := &paramsSA{}

	switch entity := entity.(type) {
	case *pb.Repository:
		params.Owner = entity.GetOwner()
		params.Repo = entity.GetName()
	case *pb.PullRequest:
		params.Owner = entity.GetRepoOwner()
		params.Repo = entity.GetRepoName()
	case *pb.Artifact:
		params.Owner = entity.GetOwner()
		params.Repo = entity.GetRepository()
	default:
		return nil, fmt.Errorf("expected repository, pull request or artifact, got %T", entity)
	}
	// TODO: Verify if this is the correct format
	packageName := fmt.Sprintf("%s/%s", params.Owner, params.Repo)
	params.Vulnerabilities = []*github.AdvisoryVulnerability{
		{
			Package: &github.VulnerabilityPackage{
				Name: &packageName,
			},
		},
	}
	// Unmarshal the existing alert metadata, if any
	if metadata != nil {
		meta := &alertMetadata{}
		err := json.Unmarshal(*metadata, meta)
		if err != nil {
			// There's nothing saved apparently, so no need to fail here, but do log the error
			logger.Debug().Msgf("error unmarshalling alert metadata: %v", err)
		} else {
			params.Metadata = meta
		}
	}
	// TODO: populate summary and description
	return params, nil
}
