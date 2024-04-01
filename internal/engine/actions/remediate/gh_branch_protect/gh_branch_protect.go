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

// Package gh_branch_protect provides the github branch protection remediation engine
package gh_branch_protect

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"text/template"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/google/go-github/v60/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/reflect/protoreflect"

	engerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// RemediateType is the type of the REST remediation engine
	RemediateType = "gh_branch_protection"
)

// GhBranchProtectRemediator keeps the status for a rule type that uses GH API to remediate branch protection
type GhBranchProtectRemediator struct {
	actionType    interfaces.ActionType
	cli           provifv1.GitHub
	patchTemplate *template.Template
}

// NewGhBranchProtectRemediator creates a new remediation engine that uses the GitHub API for branch protection
func NewGhBranchProtectRemediator(
	actionType interfaces.ActionType,
	ghp *pb.RuleType_Definition_Remediate_GhBranchProtectionType,
	pbuild *providers.ProviderBuilder,
) (*GhBranchProtectRemediator, error) {
	if actionType == "" {
		return nil, fmt.Errorf("action type cannot be empty")
	}

	patchTemplate, err := util.ParseNewTextTemplate(&ghp.Patch, "patch")
	if err != nil {
		return nil, fmt.Errorf("cannot parse patch template: %w", err)
	}

	cli, err := pbuild.GetGitHub()
	if err != nil {
		return nil, fmt.Errorf("cannot get http client: %w", err)
	}
	return &GhBranchProtectRemediator{
		actionType:    actionType,
		cli:           cli,
		patchTemplate: patchTemplate,
	}, nil
}

// PatchTemplateParams is the parameters for the REST endpoint template
type PatchTemplateParams struct {
	// Entity is the entity to be evaluated
	Entity any
	// Profile are the parameters to be used in the template
	Profile map[string]any
	// Profile are the parameters to be used in the template
	Params map[string]any
}

// Class returns the action type of the remediation engine
func (r *GhBranchProtectRemediator) Class() interfaces.ActionType {
	return r.actionType
}

// Type returns the action subtype of the remediation engine
func (_ *GhBranchProtectRemediator) Type() string {
	return RemediateType
}

// GetOnOffState returns the alert action state read from the profile
func (_ *GhBranchProtectRemediator) GetOnOffState(p *pb.Profile) interfaces.ActionOpt {
	return interfaces.ActionOptFromString(p.Remediate, interfaces.ActionOptOff)
}

// Do perform the remediation
func (r *GhBranchProtectRemediator) Do(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	remAction interfaces.ActionOpt,
	ent protoreflect.ProtoMessage,
	params interfaces.ActionsParams,
	_ *json.RawMessage,
) (json.RawMessage, error) {
	// Remediating through rest(gh_branch_protection uses REST calls) doesn't really have a turn-off behavior so
	// only proceed with the remediation if the command is to turn on the action
	if cmd != interfaces.ActionCmdOn {
		return nil, engerrors.ErrActionSkipped
	}

	retp := &PatchTemplateParams{
		Entity:  ent,
		Profile: params.GetRule().Def.AsMap(),
		Params:  params.GetRule().Params.AsMap(),
	}

	repo, ok := ent.(*pb.Repository)
	if !ok {
		return nil, fmt.Errorf("expected repository, got %T", ent)
	}

	branch, err := util.JQReadFrom[string](ctx, ".branch", params.GetRule().Params.AsMap())
	if err != nil {
		return nil, fmt.Errorf("error reading branch from params: %w", err)
	}

	// get the current protection
	res, err := r.cli.GetBranchProtection(ctx, repo.Owner, repo.Name, branch)
	if errors.Is(err, github.ErrBranchNotProtected) {
		// this will create a new branch protection using github's defaults
		// which appear quite sensible
		res = &github.Protection{}
	} else if err != nil {
		return nil, fmt.Errorf("error getting branch protection: %w", err)
	}

	req := protectionResultToRequest(res)

	var patch bytes.Buffer
	err = r.patchTemplate.Execute(&patch, retp)
	if err != nil {
		return nil, fmt.Errorf("cannot execute endpoint template: %w", err)
	}

	zerolog.Ctx(ctx).Debug().Str("patch", patch.String()).Msg("patch")

	updatedRequest, err := patchRequest(req, patch.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error patching request: %w", err)
	}

	switch remAction {
	case interfaces.ActionOptOn:
		err = r.cli.UpdateBranchProtection(ctx, repo.Owner, repo.Name, branch, updatedRequest)
	case interfaces.ActionOptDryRun:
		err = dryRun(r.cli.GetBaseURL(), repo.Owner, repo.Name, branch, updatedRequest)
	case interfaces.ActionOptOff, interfaces.ActionOptUnknown:
		err = errors.New("unexpected action")
	}
	return nil, err
}

func dryRun(baseUrl, owner, repo, branch string, req *github.ProtectionRequest) error {
	jsonReq, err := json.Marshal(req)
	if err != nil {
		// this should not be fatal
		log.Err(err).Msg("Error marshalling data")
		return fmt.Errorf("error marshalling data: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%v/%v/branches/%v/protection", owner, repo, branch)
	curlCmd, err := util.GenerateCurlCommand(http.MethodPut, baseUrl, endpoint, string(jsonReq))
	if err != nil {
		return fmt.Errorf("cannot generate curl command: %w", err)
	}

	log.Printf("run the following curl command: \n%s\n", curlCmd)
	return nil
}

func patchRequest(
	req *github.ProtectionRequest,
	patch []byte,
) (*github.ProtectionRequest, error) {
	jReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	mergedBytes, err := jsonpatch.MergePatch(jReq, patch)
	if err != nil {
		return nil, fmt.Errorf("error merging patch: %w", err)
	}

	merged := &github.ProtectionRequest{}
	err = json.Unmarshal(mergedBytes, merged)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling merged request: %w", err)
	}

	return merged, nil
}

func protectionResultToRequest(res *github.Protection) *github.ProtectionRequest {
	req := &github.ProtectionRequest{
		Restrictions:               branchRestrictionResponseToRequest(res.Restrictions),
		RequiredPullRequestReviews: pullRequestReviewsResponseToRequest(res.RequiredPullRequestReviews),
		RequiredStatusChecks:       res.RequiredStatusChecks,
	}

	if req.RequiredStatusChecks != nil {
		if req.RequiredStatusChecks.Checks != nil && len(req.GetRequiredStatusChecks().GetContexts()) > 0 {
			// if both are set, the API will return an error as Contexts is now deprecated
			// but at the same time the API does return both fields, so we filter the deprecated
			// one manually
			req.RequiredStatusChecks.Contexts = nil
		}
	}

	if res.EnforceAdmins != nil {
		req.EnforceAdmins = res.EnforceAdmins.Enabled
	}

	if res.AllowForkSyncing != nil {
		req.AllowForkSyncing = res.AllowForkSyncing.Enabled
	}
	if res.LockBranch != nil {
		req.LockBranch = res.LockBranch.Enabled
	}
	if res.BlockCreations != nil {
		req.BlockCreations = res.BlockCreations.Enabled
	}
	if res.RequiredConversationResolution != nil {
		req.RequiredConversationResolution = github.Bool(res.RequiredConversationResolution.Enabled)
	}
	if res.AllowDeletions != nil {
		req.AllowDeletions = github.Bool(res.AllowDeletions.Enabled)
	}
	if res.AllowForcePushes != nil {
		req.AllowForcePushes = github.Bool(res.AllowForcePushes.Enabled)
	}
	if res.RequireLinearHistory != nil {
		req.RequireLinearHistory = github.Bool(res.RequireLinearHistory.Enabled)
	}

	return req
}

func branchRestrictionResponseToRequest(res *github.BranchRestrictions) *github.BranchRestrictionsRequest {
	if res == nil {
		return nil
	}
	return &github.BranchRestrictionsRequest{
		Users: userSliceToStringSlice(res.Users),
		Teams: sluggerSliceToStringSlice(res.Teams),
		Apps:  sluggerSliceToStringSlice(res.Teams),
	}
}

func pullRequestReviewsResponseToRequest(res *github.PullRequestReviewsEnforcement) *github.PullRequestReviewsEnforcementRequest {
	if res == nil {
		return nil
	}
	req := &github.PullRequestReviewsEnforcementRequest{
		DismissStaleReviews:          res.DismissStaleReviews,
		RequireCodeOwnerReviews:      res.RequireCodeOwnerReviews,
		RequiredApprovingReviewCount: res.RequiredApprovingReviewCount,
		RequireLastPushApproval:      github.Bool(res.RequireLastPushApproval),
	}

	if res.BypassPullRequestAllowances != nil {
		req.BypassPullRequestAllowancesRequest = &github.BypassPullRequestAllowancesRequest{
			Users: userSliceToStringSlice(res.BypassPullRequestAllowances.Users),
			Teams: sluggerSliceToStringSlice(res.BypassPullRequestAllowances.Teams),
			Apps:  sluggerSliceToStringSlice(res.BypassPullRequestAllowances.Apps),
		}
	}

	if res.DismissalRestrictions != nil {
		users := userSliceToStringSlice(res.DismissalRestrictions.Users)
		teams := sluggerSliceToStringSlice(res.DismissalRestrictions.Teams)
		apps := sluggerSliceToStringSlice(res.DismissalRestrictions.Apps)
		req.DismissalRestrictionsRequest = &github.DismissalRestrictionsRequest{
			Users: &users,
			Teams: &teams,
			Apps:  &apps,
		}
	}

	return req
}

func userSliceToStringSlice(users []*github.User) []string {
	// nil slice throws error, explicitly create an empty slice
	userStrs := make([]string, 0) // using make
	for _, user := range users {
		userStrs = append(userStrs, user.GetLogin())
	}
	return userStrs
}

type slugger interface {
	GetSlug() string
}

func sluggerSliceToStringSlice[S slugger](items []S) []string {
	// nil slice throws error, explicitly create an empty slice
	strs := make([]string, 0)
	for _, item := range items {
		strs = append(strs, item.GetSlug())
	}
	return strs
}
