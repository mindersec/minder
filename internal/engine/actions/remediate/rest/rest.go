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

// Package rest provides the REST remediation engine
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v61/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/reflect/protoreflect"

	engerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// RemediateType is the type of the REST remediation engine
	RemediateType = "rest"

	// EndpointBytesLimit is the maximum number of bytes for the endpoint
	EndpointBytesLimit = 1024

	// BodyBytesLimit is the maximum number of bytes for the body
	BodyBytesLimit = 5120
)

// Remediator keeps the status for a rule type that uses REST remediation
type Remediator struct {
	actionType       interfaces.ActionType
	method           string
	cli              provifv1.REST
	endpointTemplate *util.SafeTemplate
	bodyTemplate     *util.SafeTemplate
}

// NewRestRemediate creates a new REST rule data ingest engine
func NewRestRemediate(actionType interfaces.ActionType, restCfg *pb.RestType, cli provifv1.REST) (*Remediator, error) {
	if actionType == "" {
		return nil, fmt.Errorf("action type cannot be empty")
	}

	endpointTmpl, err := util.NewSafeTextTemplate(&restCfg.Endpoint, "endpoint")
	if err != nil {
		return nil, fmt.Errorf("cannot parse endpoint template: %w", err)
	}

	var bodyTmpl *util.SafeTemplate
	if restCfg.Body != nil {
		bodyTmpl, err = util.NewSafeTextTemplate(restCfg.Body, "body")
		if err != nil {
			return nil, fmt.Errorf("cannot parse body template: %w", err)
		}
	}

	method := util.HttpMethodFromString(restCfg.Method, http.MethodPatch)

	return &Remediator{
		cli:              cli,
		actionType:       actionType,
		method:           method,
		endpointTemplate: endpointTmpl,
		bodyTemplate:     bodyTmpl,
	}, nil
}

// EndpointTemplateParams is the parameters for the REST endpoint template
type EndpointTemplateParams struct {
	// Entity is the entity to be evaluated
	Entity any
	// Profile is the parameters to be used in the template
	Profile map[string]any
	// Params are the rule instance parameters
	Params map[string]any
}

// Class returns the action type of the remediation engine
func (r *Remediator) Class() interfaces.ActionType {
	return r.actionType
}

// Type returns the action subtype of the remediation engine
func (_ *Remediator) Type() string {
	return RemediateType
}

// GetOnOffState returns the alert action state read from the profile
func (_ *Remediator) GetOnOffState(p *pb.Profile) interfaces.ActionOpt {
	return interfaces.ActionOptFromString(p.Remediate, interfaces.ActionOptOff)
}

// Do perform the remediation
func (r *Remediator) Do(
	ctx context.Context,
	cmd interfaces.ActionCmd,
	setting interfaces.ActionOpt,
	entity protoreflect.ProtoMessage,
	params interfaces.ActionsParams,
	_ *json.RawMessage,
) (json.RawMessage, error) {
	// Remediating through rest doesn't really have a turn-off behavior so
	// only proceed with the remediation if the command is to turn on the action
	if cmd != interfaces.ActionCmdOn {
		return nil, engerrors.ErrActionSkipped
	}

	retp := &EndpointTemplateParams{
		Entity:  entity,
		Profile: params.GetRule().Def.AsMap(),
		Params:  params.GetRule().Params.AsMap(),
	}

	endpoint := new(bytes.Buffer)
	if err := r.endpointTemplate.Execute(ctx, endpoint, retp, EndpointBytesLimit); err != nil {
		return nil, fmt.Errorf("cannot execute endpoint template: %w", err)
	}

	body := new(bytes.Buffer)
	if r.bodyTemplate != nil {
		if err := r.bodyTemplate.Execute(ctx, body, retp, BodyBytesLimit); err != nil {
			return nil, fmt.Errorf("cannot execute endpoint template: %w", err)
		}
	}

	zerolog.Ctx(ctx).Debug().
		Msgf("remediating with endpoint: [%s] and body [%+v]", endpoint.String(), body.String())

	var err error
	switch setting {
	case interfaces.ActionOptOn:
		err = r.run(ctx, endpoint.String(), body.Bytes())
	case interfaces.ActionOptDryRun:
		err = r.dryRun(ctx, endpoint.String(), body.String())
	case interfaces.ActionOptOff, interfaces.ActionOptUnknown:
		err = errors.New("unexpected action")
	}
	return nil, err
}

func (r *Remediator) run(ctx context.Context, endpoint string, body []byte) error {
	// create an empty map, not a nil map to avoid passing nil to NewRequest
	bodyJson := make(map[string]any)

	if len(body) > 0 {
		err := json.Unmarshal(body, &bodyJson)
		if err != nil {
			return fmt.Errorf("cannot unmarshal body: %w", err)
		}
	}

	req, err := r.cli.NewRequest(r.method, endpoint, bodyJson)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}

	resp, err := r.cli.Do(ctx, req)
	if err != nil {
		var respErr *github.ErrorResponse
		if errors.As(err, &respErr) {
			zerolog.Ctx(ctx).Error().Msgf("Error message: %v", respErr.Message)
			for _, e := range respErr.Errors {
				zerolog.Ctx(ctx).Error().Msgf("Field: %s, Message: %s", e.Field, e.Message)
			}
		}
		return fmt.Errorf("cannot make request: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("cannot close response body: %v", err)
		}
	}()
	// Translate the http status code response to an error
	if engerrors.HTTPErrorCodeToErr(resp.StatusCode) != nil {
		return engerrors.NewErrActionFailed("remediation failed: %s", err)
	}
	return nil
}

func (r *Remediator) dryRun(ctx context.Context, endpoint, body string) error {
	curlCmd, err := util.GenerateCurlCommand(ctx, r.method, r.cli.GetBaseURL(), endpoint, body)
	if err != nil {
		return fmt.Errorf("cannot generate curl command: %w", err)
	}

	log.Printf("run the following curl command: \n%s\n", curlCmd)
	return nil
}
