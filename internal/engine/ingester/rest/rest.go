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

// Package rest provides the REST rule data ingest engine
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"google.golang.org/protobuf/reflect/protoreflect"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

const (
	// RestRuleDataIngestType is the type of the REST rule data ingest engine
	RestRuleDataIngestType = "rest"
)

// Ingestor is the engine for a rule type that uses REST data ingest
type Ingestor struct {
	restCfg          *pb.RestType
	cli              ghclient.RestAPI
	endpointTemplate *template.Template
	method           string
}

// NewRestRuleDataIngest creates a new REST rule data ingest engine
func NewRestRuleDataIngest(
	restCfg *pb.RestType,
	cli ghclient.RestAPI,
) (*Ingestor, error) {
	if len(restCfg.Endpoint) == 0 {
		return nil, fmt.Errorf("missing endpoint")
	}

	tmpl := template.New("path")
	tmpl, err := tmpl.Parse(restCfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("cannot parse endpoint template: %w", err)
	}

	method := strings.ToUpper(restCfg.Method)
	if len(method) == 0 {
		method = http.MethodGet
	}

	return &Ingestor{
		restCfg:          restCfg,
		cli:              cli,
		endpointTemplate: tmpl,
		method:           method,
	}, nil
}

// EndpointTemplateParams is the parameters for the REST endpoint template
type EndpointTemplateParams struct {
	// Entity is the entity to be evaluated
	Entity any
	// Params are the parameters to be used in the template
	Params map[string]any
}

// Ingest calls the REST endpoint and returns the data
func (rdi *Ingestor) Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (any, error) {
	endpoint := new(bytes.Buffer)
	retp := &EndpointTemplateParams{
		Entity: ent,
		Params: params,
	}

	if err := rdi.endpointTemplate.Execute(endpoint, retp); err != nil {
		return nil, fmt.Errorf("cannot execute endpoint template: %w", err)
	}

	req, err := rdi.cli.NewRequest(rdi.method, endpoint.String(), rdi.restCfg.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}

	bodyBuf := new(bytes.Buffer)
	_, err = rdi.cli.Do(ctx, req, bodyBuf)
	if err != nil {
		return nil, fmt.Errorf("cannot make request: %w", err)
	}

	var data any
	data = bodyBuf

	if rdi.restCfg.Parse == "json" {
		var jsonData any
		dec := json.NewDecoder(bodyBuf)
		if err := dec.Decode(&jsonData); err != nil {
			return nil, fmt.Errorf("cannot decode json: %w", err)
		}

		data = jsonData
	}

	return data, nil
}
