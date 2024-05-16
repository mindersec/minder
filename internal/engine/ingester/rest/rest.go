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

// Package rest provides the REST rule data ingest engine
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/google/go-github/v61/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/reflect/protoreflect"

	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// RestRuleDataIngestType is the type of the REST rule data ingest engine
	RestRuleDataIngestType = "rest"

	// MaxBytesLimit is the maximum number of bytes to read from the response body
	// We limit to 1MB to prevent abuse
	MaxBytesLimit int64 = 1 << 20
)

type ingestorFallback struct {
	// httpCode is the HTTP status code to return
	httpCode int
	// Body is the body to return
	body string
}

// Ingestor is the engine for a rule type that uses REST data ingest
type Ingestor struct {
	restCfg          *pb.RestType
	cli              provifv1.REST
	endpointTemplate *template.Template
	method           string
	fallback         []ingestorFallback
}

// NewRestRuleDataIngest creates a new REST rule data ingest engine
func NewRestRuleDataIngest(
	restCfg *pb.RestType,
	cli provifv1.REST,
) (*Ingestor, error) {
	if len(restCfg.Endpoint) == 0 {
		return nil, fmt.Errorf("missing endpoint")
	}

	tmpl, err := util.ParseNewTextTemplate(&restCfg.Endpoint, "endpoint")
	if err != nil {
		return nil, fmt.Errorf("cannot parse endpoint template: %w", err)
	}

	method := util.HttpMethodFromString(restCfg.Method, http.MethodGet)

	fallback := make([]ingestorFallback, len(restCfg.Fallback))
	for _, fb := range restCfg.Fallback {
		fb := fb
		fallback = append(fallback, ingestorFallback{
			httpCode: int(fb.HttpCode),
			body:     fb.Body,
		})
	}

	return &Ingestor{
		restCfg:          restCfg,
		cli:              cli,
		endpointTemplate: tmpl,
		method:           method,
		fallback:         fallback,
	}, nil
}

// EndpointTemplateParams is the parameters for the REST endpoint template
type EndpointTemplateParams struct {
	// Entity is the entity to be evaluated
	Entity any
	// Params are the parameters to be used in the template
	Params map[string]any
}

// GetType returns the type of the REST rule data ingest engine
func (*Ingestor) GetType() string {
	return RestRuleDataIngestType
}

// GetConfig returns the config for the REST rule data ingest engine
func (rdi *Ingestor) GetConfig() protoreflect.ProtoMessage {
	return rdi.restCfg
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

	respRdr, err := rdi.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cannot do request: %w", err)
	}

	defer func() {
		if err := respRdr.Close(); err != nil {
			log.Printf("cannot close response body: %v", err)
		}
	}()

	data, err := rdi.parseBody(respRdr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse body: %w", err)
	}

	return &engif.Result{
		Object: data,
	}, nil
}

func (rdi *Ingestor) doRequest(ctx context.Context, req *http.Request) (io.ReadCloser, error) {
	resp, err := rdi.cli.Do(ctx, req)
	if err == nil {
		// Early-exit on success
		return resp.Body, nil
	}

	if fallbackBody := errorToFallback(err, rdi.fallback); fallbackBody != nil {
		// the go-github REST API has a funny way of returning HTTP status codes,
		// on a non-200 status it will return a github.ErrorResponse
		// whereas the standard library will return nil error and the HTTP status code in the response
		return fallbackBody, nil
	}

	return nil, fmt.Errorf("cannot make request: %w", err)
}

func errorToFallback(err error, fallback []ingestorFallback) io.ReadCloser {
	var respErr *github.ErrorResponse
	if errors.As(err, &respErr) {
		if respErr.Response != nil {
			return httpStatusToFallback(respErr.Response.StatusCode, fallback)
		}
	}

	return nil
}

func httpStatusToFallback(httpStatus int, fallback []ingestorFallback) io.ReadCloser {
	for _, fb := range fallback {
		if fb.httpCode == httpStatus {
			zerolog.Ctx(context.Background()).Debug().Msgf("falling back to body [%s]", fb.body)
			return io.NopCloser(strings.NewReader(fb.body))
		}
	}

	return nil
}

func (rdi *Ingestor) parseBody(body io.Reader) (any, error) {
	var data any
	var err error

	if body == nil {
		return nil, nil
	}

	lr := io.LimitReader(body, MaxBytesLimit)

	if rdi.restCfg.Parse == "json" {
		var jsonData any
		dec := json.NewDecoder(lr)
		if err := dec.Decode(&jsonData); err != nil {
			return nil, fmt.Errorf("cannot decode json: %w", err)
		}

		data = jsonData
	} else {
		data, err = io.ReadAll(lr)
		if err != nil {
			return nil, fmt.Errorf("cannot read response body: %w", err)
		}
	}

	return data, nil
}
