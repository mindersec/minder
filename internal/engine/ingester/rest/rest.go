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

// Package rest provides the REST rule data ingest engine
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"

	"google.golang.org/protobuf/reflect/protoreflect"

	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
	provifv1 "github.com/stacklok/mediator/pkg/providers/v1"
)

const (
	// RestRuleDataIngestType is the type of the REST rule data ingest engine
	RestRuleDataIngestType = "rest"
)

// Ingestor is the engine for a rule type that uses REST data ingest
type Ingestor struct {
	restCfg          *pb.RestType
	cli              provifv1.REST
	endpointTemplate *template.Template
	method           string
}

// NewRestRuleDataIngest creates a new REST rule data ingest engine
func NewRestRuleDataIngest(
	restCfg *pb.RestType,
	pbuild *providers.ProviderBuilder,
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

	cli, err := pbuild.GetHTTP(context.Background())
	if err != nil {
		return nil, fmt.Errorf("cannot get http client: %w", err)
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
func (rdi *Ingestor) Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (*engif.Result, error) {
	endpoint := new(bytes.Buffer)
	retp := &EndpointTemplateParams{
		Entity: ent,
		Params: params,
	}

	if err := rdi.endpointTemplate.Execute(endpoint, retp); err != nil {
		return nil, fmt.Errorf("cannot execute endpoint template: %w", err)
	}

	// create string buffer
	var bodyr io.Reader
	if rdi.restCfg.Body != nil {
		bodyr = strings.NewReader(*rdi.restCfg.Body)
	}

	req, err := rdi.cli.NewRequest(rdi.method, endpoint.String(), bodyr)
	if err != nil {
		return nil, fmt.Errorf("cannot create request: %w", err)
	}

	resp, err := rdi.cli.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cannot make request: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("cannot close response body: %v", err)
		}
	}()

	var data any

	if rdi.restCfg.Parse == "json" {
		var jsonData any
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&jsonData); err != nil {
			return nil, fmt.Errorf("cannot decode json: %w", err)
		}

		data = jsonData
	} else {
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read response body: %w", err)
		}
	}

	return &engif.Result{
		Object: data,
	}, nil
}
