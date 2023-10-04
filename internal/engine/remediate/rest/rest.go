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
	"log"
	"net/http"
	"text/template"

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
	// RemediateType is the type of the REST remediation engine
	RemediateType = "rest"
)

var (
	// ErrUnauthorized is returned when the remediation request is unauthorized
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden is returned when the remediation request is forbidden
	ErrForbidden = errors.New("forbidden")
	// ErrClientError is returned when the remediation request is a client error
	ErrClientError = errors.New("client error")
	// ErrServerError is returned when the remediation request is a server error
	ErrServerError = errors.New("server error")
	// ErrOther is returned when the remediation request is an other error
	ErrOther = errors.New("other error")
)

// Remediator keeps the status for a rule type that uses REST remediation
type Remediator struct {
	cli provifv1.REST

	method           string
	endpointTemplate *template.Template
	bodyTemplate     *template.Template
}

// NewRestRemediate creates a new REST rule data ingest engine
func NewRestRemediate(
	restCfg *pb.RestType,
	pbuild *providers.ProviderBuilder,
) (*Remediator, error) {
	endpointTmpl, err := util.ParseNewTemplate(&restCfg.Endpoint, "endpoint")
	if err != nil {
		return nil, fmt.Errorf("cannot parse endpoint template: %w", err)
	}

	bodyTmpl, err := util.ParseNewTemplate(restCfg.Body, "body")
	if err != nil {
		return nil, fmt.Errorf("cannot parse body template: %w", err)
	}

	method := util.HttpMethodFromString(restCfg.Method, http.MethodPatch)

	cli, err := pbuild.GetHTTP(context.Background())
	if err != nil {
		return nil, fmt.Errorf("cannot get http client: %w", err)
	}

	return &Remediator{
		cli:              cli,
		method:           method,
		endpointTemplate: endpointTmpl,
		bodyTemplate:     bodyTmpl,
	}, nil
}

// EndpointTemplateParams is the parameters for the REST endpoint template
type EndpointTemplateParams struct {
	// Entity is the entity to be evaluated
	Entity any
	// Policy are the parameters to be used in the template
	Policy map[string]any
}

// Remediate actually performs the remediation
func (r *Remediator) Remediate(
	ctx context.Context,
	remAction interfaces.RemediateActionOpt,
	ent protoreflect.ProtoMessage,
	pol map[string]any,
) error {
	retp := &EndpointTemplateParams{
		Entity: ent,
		Policy: pol,
	}

	endpoint := new(bytes.Buffer)
	if err := r.endpointTemplate.Execute(endpoint, retp); err != nil {
		return fmt.Errorf("cannot execute endpoint template: %w", err)
	}

	body := new(bytes.Buffer)
	if err := r.bodyTemplate.Execute(body, retp); err != nil {
		return fmt.Errorf("cannot execute endpoint template: %w", err)
	}

	zerolog.Ctx(ctx).Debug().
		Msgf("remediating with endpoint: [%s] and body [%+v]", endpoint.String(), body.String())

	var err error
	switch remAction {
	case interfaces.ActionOptOn:
		err = r.run(ctx, endpoint.String(), body.String())
	case interfaces.ActionOptDryRun:
		err = r.dryRun(endpoint.String(), body.String())
	case interfaces.ActionOptOff, interfaces.ActionOptUnknown:
		err = errors.New("unexpected action")
	}
	return err
}

func (r *Remediator) run(ctx context.Context, endpoint string, body string) error {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(body), &result)
	if err != nil {
		return fmt.Errorf("cannot unmarshal body: %w", err)
	}

	req, err := r.cli.NewRequest(r.method, endpoint, result)
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

	return httpErrorCodeToErr(resp.StatusCode)
}

func (r *Remediator) dryRun(endpoint, body string) error {
	curlCmd, err := util.GenerateCurlCommand(r.method, r.cli.GetBaseURL(), endpoint, body)
	if err != nil {
		return fmt.Errorf("cannot generate curl command: %w", err)
	}

	log.Printf("run the following curl command: \n%s\n", curlCmd)
	return nil
}

func httpErrorCodeToErr(httpCode int) error {
	var err = ErrOther

	switch {
	case httpCode >= 200 && httpCode < 300:
		err = nil
	case httpCode == 401:
		err = ErrUnauthorized
	case httpCode == 403:
		err = ErrForbidden
	case httpCode >= 400 && httpCode < 500:
		err = ErrClientError
	case httpCode >= 500:
		err = ErrServerError
	}

	if err != nil {
		return enginerr.NewErrRemediationFailed("remediation failed: %s", err)
	}

	return nil
}
