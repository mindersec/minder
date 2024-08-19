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
	"errors"
	"fmt"
	htmltemplate "html/template"
	"strings"

	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/profiles/models"
	"google.golang.org/protobuf/reflect/protoreflect"

	enginerr "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// AlertType is the type of the security advisory alert engine
	AlertType                = "security_advisory"
	tmplSummaryName          = "summary"
	tmplSummary              = `minder: profile {{.Profile}} failed with rule {{.Rule}}`
	tmplDescriptionNameNoRem = "description_no_remediate"
	tmplDescriptionNameRem   = "description"
	// nolint:lll
	tmplPart1Top = `
Minder has detected a potential security exposure in your repository - **{{.Repository}}**.
This exposure has been classified with a severity level of **{{.Severity}}**, as per the configuration defined in the **{{.Rule}}** rule type.

The purpose of this security advisory is to alert you to the presence of this exposure. Please note that this advisory has been automatically generated as a result of having the alert feature enabled within the **{{.Profile}}** profile.

This advisory will be automatically closed once the issue associated with the **{{.Rule}}** rule is resolved.
`
	// nolint:lll
	tmplPart2MiddleNoRem = `
**Remediation**

To address this security exposure, we recommend taking the following actions:

1. Enable the auto-remediate feature within the **{{.Profile}}** profile by following the [Minder documentation](https://minder-docs.stacklok.dev/understand/remediation). This will allow Minder to automatically remediate this and other vulnerabilities in the future, provided that a remediation action is available for the given rule type. In the case of the **{{.Rule}}** rule type, the remediation action is **{{.RuleRemediation}}**.
2. Alternatively, you can manually address this issue by following the guidance provided below.
`
	// nolint:lll
	tmplPart2MiddleRem = `
**Remediation**

To address this security exposure, we recommend taking the following actions:

1. Since you've turned on the remediate feature in your profile, Minder may have already taken steps to address this issue. Please check for pending remediation actions, such as open pull requests, that require your review and approval.
2. In case Minder was not able to remediate this automatically, please refer to the guidance below to resolve the issue manually.
`
	// nolint:lll
	tmplPart3Bottom = `
**Guidance**

{{.Guidance}}

**Details**

- Profile: {{.Profile}}
- Rule: {{.Rule}}
{{if (ne .Name .Rule) -}}
- Name: {{.Name}}
{{end -}}
- Repository: {{.Repository}}
- Severity: {{.Severity}}

**About**

If you have any questions or believe that this evaluation is incorrect, please don't hesitate to reach out to the Minder team at info@stacklok.com.
`
)

// Alert is the structure backing the security-advisory alert action
type Alert struct {
	actionType           interfaces.ActionType
	cli                  provifv1.GitHub
	ruleType             *pb.RuleType
	saCfg                *pb.RuleType_Definition_Alert_AlertTypeSA
	summaryTmpl          *htmltemplate.Template
	descriptionTmpl      *htmltemplate.Template
	descriptionNoRemTmpl *htmltemplate.Template
}

type paramsSA struct {
	// Used by the template
	Template        templateParamsSA
	Owner           string
	Repo            string
	Summary         string
	Description     string
	Vulnerabilities []*github.AdvisoryVulnerability
	Metadata        *alertMetadata
	prevStatus      *db.ListRuleEvaluationsByProfileIdRow
}

type templateParamsSA struct {
	Profile         string
	Rule            string
	Repository      string
	Severity        string
	Guidance        string
	RuleRemediation string
	Name            string
}

type alertMetadata struct {
	ID string `json:"ghsa_id,omitempty"`
}

// NewSecurityAdvisoryAlert creates a new security-advisory alert action
func NewSecurityAdvisoryAlert(
	actionType interfaces.ActionType,
	ruleType *pb.RuleType,
	saCfg *pb.RuleType_Definition_Alert_AlertTypeSA,
	cli provifv1.GitHub,
) (*Alert, error) {
	if actionType == "" {
		return nil, fmt.Errorf("action type cannot be empty")
	}
	// Parse the templates for summary and description
	sumT, err := htmltemplate.New(tmplSummaryName).Option("missingkey=error").Parse(tmplSummary)
	if err != nil {
		return nil, fmt.Errorf("cannot parse summary template: %w", err)
	}
	descriptionTmplNoRemStr := strings.Join([]string{tmplPart1Top, tmplPart2MiddleNoRem, tmplPart3Bottom}, "\n")
	descNoRemT, err := htmltemplate.New(tmplDescriptionNameNoRem).Option("missingkey=error").Parse(descriptionTmplNoRemStr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse description template: %w", err)
	}
	descriptionTmplStr := strings.Join([]string{tmplPart1Top, tmplPart2MiddleRem, tmplPart3Bottom}, "\n")
	descT, err := htmltemplate.New(tmplDescriptionNameRem).Option("missingkey=error").Parse(descriptionTmplStr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse description template: %w", err)
	}

	// Create the alert action
	return &Alert{
		actionType:           actionType,
		cli:                  cli,
		ruleType:             ruleType,
		saCfg:                saCfg,
		summaryTmpl:          sumT,
		descriptionTmpl:      descT,
		descriptionNoRemTmpl: descNoRemT,
	}, nil
}

// Class returns the action type of the security-advisory engine
func (alert *Alert) Class() interfaces.ActionType {
	return alert.actionType
}

// Type returns the action subtype of the remediation engine
func (_ *Alert) Type() string {
	return AlertType
}

// GetOnOffState returns the alert action state read from the profile
func (_ *Alert) GetOnOffState(actionOpt models.ActionOpt) models.ActionOpt {
	return models.ActionOptOrDefault(actionOpt, models.ActionOptOff)
}

// Do alerts through security advisory
func (alert *Alert) Do(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	setting models.ActionOpt,
	entity protoreflect.ProtoMessage,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (json.RawMessage, error) {
	// Get the parameters for the security advisory - owner, repo, etc.
	p, err := alert.getParamsForSecurityAdvisory(ctx, entity, params, metadata)
	if err != nil {
		return nil, fmt.Errorf("error extracting details: %w", err)
	}

	// Process the command based on the action setting
	switch setting {
	case models.ActionOptOn:
		return alert.run(ctx, p, cmd)
	case models.ActionOptDryRun:
		return alert.runDry(ctx, p, cmd)
	case models.ActionOptOff, models.ActionOptUnknown:
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
			params.Template.Severity,
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
		logger.Info().Str("ghsa_id", id).Msg("security advisory opened")
		return newMeta, nil
	// Close a security advisory
	case interfaces.ActionCmdOff:
		if params.Metadata == nil || params.Metadata.ID == "" {
			// We cannot do anything without the GHSA_ID, so we assume that closing this is a success
			return nil, fmt.Errorf("no security advisory GHSA_ID provided: %w", enginerr.ErrActionTurnedOff)
		}
		err := alert.cli.CloseSecurityAdvisory(ctx, params.Owner, params.Repo, params.Metadata.ID)
		if err != nil {
			if errors.Is(err, enginerr.ErrNotFound) {
				// There's no security advisory with such GHSA_ID anymore (perhaps it was closed manually).
				// We exit by stating that the action was turned off.
				return nil, fmt.Errorf("security advisory already closed: %w, %w", err, enginerr.ErrActionTurnedOff)
			}
			return nil, fmt.Errorf("error closing security advisory: %w, %w", err, enginerr.ErrActionFailed)
		}
		logger.Info().Str("ghsa_id", params.Metadata.ID).Msg("security advisory closed")
		// Success - return ErrActionTurnedOff to indicate the action was successful
		return nil, fmt.Errorf("%s : %w", alert.Class(), enginerr.ErrActionTurnedOff)
	case interfaces.ActionCmdDoNothing:
		// Return the previous alert status.
		return alert.runDoNothing(ctx, params)
	}
	return nil, enginerr.ErrActionSkipped
}

// runDry runs the security advisory action in dry run mode
func (alert *Alert) runDry(ctx context.Context, params *paramsSA, cmd interfaces.ActionCmd) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx)

	// Process the command
	switch cmd {
	// Open a security advisory
	case interfaces.ActionCmdOn:
		endpoint := fmt.Sprintf("repos/%v/%v/security-advisories", params.Owner, params.Repo)
		body := ""
		curlCmd, err := util.GenerateCurlCommand(ctx, "POST", alert.cli.GetBaseURL(), endpoint, body)
		if err != nil {
			return nil, fmt.Errorf("cannot generate curl command: %w", err)
		}
		logger.Info().Msgf("run the following curl command to open a security-advisory: \n%s\n", curlCmd)
		return nil, nil
	// Close a security advisory
	case interfaces.ActionCmdOff:
		if params.Metadata == nil || params.Metadata.ID == "" {
			// We cannot do anything without the GHSA_ID, so we assume that closing this is a success
			return nil, fmt.Errorf("no security advisory GHSA_ID provided: %w", enginerr.ErrActionTurnedOff)
		}
		endpoint := fmt.Sprintf("repos/%v/%v/security-advisories/%v",
			params.Owner, params.Repo, params.Metadata.ID)
		body := "{\"state\": \"closed\"}"
		curlCmd, err := util.GenerateCurlCommand(ctx, "PATCH", alert.cli.GetBaseURL(), endpoint, body)
		if err != nil {
			return nil, fmt.Errorf("cannot generate curl command to close a security-adivsory: %w", err)
		}
		logger.Info().Msgf("run the following curl command: \n%s\n", curlCmd)
	case interfaces.ActionCmdDoNothing:
		// Return the previous alert status.
		return alert.runDoNothing(ctx, params)

	}
	return nil, enginerr.ErrActionSkipped
}

// getParamsForSecurityAdvisory extracts the details from the entity
func (alert *Alert) getParamsForSecurityAdvisory(
	ctx context.Context,
	entity protoreflect.ProtoMessage,
	params interfaces.ActionsParams,
	metadata *json.RawMessage,
) (*paramsSA, error) {
	logger := zerolog.Ctx(ctx)
	result := &paramsSA{
		prevStatus: params.GetEvalStatusFromDb(),
	}

	// Get the owner and repo from the entity
	switch entity := entity.(type) {
	case *pb.Repository:
		result.Owner = entity.GetOwner()
		result.Repo = entity.GetName()
	case *pb.PullRequest:
		result.Owner = entity.GetRepoOwner()
		result.Repo = entity.GetRepoName()
	case *pb.Artifact:
		result.Owner = entity.GetOwner()
		result.Repo = entity.GetRepository()
	default:
		return nil, fmt.Errorf("expected repository, pull request or artifact, got %T", entity)
	}
	result.Template.Repository = fmt.Sprintf("%s/%s", result.Owner, result.Repo)
	ecosystem := "other"
	result.Vulnerabilities = []*github.AdvisoryVulnerability{
		{
			Package: &github.VulnerabilityPackage{
				Name:      &result.Template.Repository,
				Ecosystem: &ecosystem,
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
			result.Metadata = meta
		}
	}
	// Process the summary and description templates
	// Get the severity
	result.Template.Severity = alert.getSeverityString()
	// Get the guidance
	result.Template.Guidance = alert.ruleType.Guidance
	// Get the rule type name
	result.Template.Rule = alert.ruleType.Name
	// Get the profile name
	result.Template.Profile = params.GetProfile().Name
	// Get the rule name
	result.Template.Name = params.GetRule().Name

	// Check if remediation is available for the rule type
	if alert.ruleType.Def.Remediate != nil {
		result.Template.RuleRemediation = "already available"
	} else {
		result.Template.RuleRemediation = "not available yet"
	}
	var summaryStr strings.Builder
	err := alert.summaryTmpl.Execute(&summaryStr, result.Template)
	if err != nil {
		return nil, fmt.Errorf("error executing summary template: %w", err)
	}
	result.Summary = summaryStr.String()

	var descriptionStr strings.Builder
	// Get the description template depending if remediation is available
	if params.GetProfile().ActionConfig.Remediate == models.ActionOptOn {
		err = alert.descriptionTmpl.Execute(&descriptionStr, result.Template)
	} else {
		err = alert.descriptionNoRemTmpl.Execute(&descriptionStr, result.Template)
	}
	if err != nil {
		return nil, fmt.Errorf("error executing description template: %w", err)
	}
	result.Description = descriptionStr.String()
	return result, nil
}

func (alert *Alert) getSeverityString() string {
	if alert.saCfg.Severity == "" {
		ruleSev := alert.ruleType.Severity.GetValue().Enum().AsString()
		if ruleSev == "info" || ruleSev == "unknown" {
			return "low"
		}
		return ruleSev
	}

	return alert.saCfg.Severity
}

// runDoNothing returns the previous alert status
func (_ *Alert) runDoNothing(ctx context.Context, params *paramsSA) (json.RawMessage, error) {
	logger := zerolog.Ctx(ctx).With().Str("repo", params.Repo).Logger()

	logger.Debug().Msg("Running do nothing")

	// Return the previous alert status.
	err := enginerr.AlertStatusAsError(params.prevStatus)
	// If there is a valid alert metadata, return it too
	if params.prevStatus != nil && params.prevStatus.AlertMetadata.Valid {
		return params.prevStatus.AlertMetadata.RawMessage, err
	}
	// If there is no alert metadata, return nil as the metadata and the error
	return nil, err
}
