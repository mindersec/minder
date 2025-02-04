// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog"
	"github.com/santhosh-tekuri/jsonschema/v6"
	uritemplate "github.com/std-uritemplate/std-uritemplate/go/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/engine/eval/rego"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/schemaupdate"
	"github.com/mindersec/minder/internal/util/schemavalidate"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

const (
	// MaxBytesLimit is the maximum number of bytes to read from the response body
	// We limit to 1MB to prevent abuse
	MaxBytesLimit int64 = 1 << 20
)

var (
	metricsInit sync.Once

	dataSourceRequestCounter   metric.Int64Counter
	dataSourceLatencyHistogram metric.Int64Histogram
)

type restHandler struct {
	rawInputSchema *structpb.Struct
	inputSchema    *jsonschema.Schema
	endpointTmpl   string
	method         string
	// used only to allow requests to localhost during tests
	testOnlyTransport http.RoundTripper
	// contains the request body or the key
	body          string
	bodyFromInput bool
	headers       map[string]string
	parse         string
	// TODO implement fallback
	// TODO implement auth
}

func initMetrics() {
	metricsInit.Do(func() {
		meter := otel.Meter("minder")
		var err error
		dataSourceRequestCounter, err = meter.Int64Counter(
			"datasource.rest.request",
			metric.WithDescription("Total number of data source requests issued"),
		)
		if err != nil {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Creating counter for data source requests failed")
		}
		dataSourceLatencyHistogram, err = meter.Int64Histogram(
			"datasource.rest.latency",
			metric.WithDescription("Latency of data source requests in milliseconds"),
		)
		if err != nil {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Creating histogram for data source requests failed")
		}
	})
}

func newHandlerFromDef(def *minderv1.RestDataSource_Def) (*restHandler, error) {
	if def == nil {
		return nil, errors.New("rest data source handler definition is nil")
	}

	// schema may be nil
	schema, err := schemavalidate.CompileSchemaFromPB(def.GetInputSchema())
	if err != nil {
		return nil, err
	}

	bodyFromInput, body := parseRequestBodyConfig(def)

	initMetrics()

	return &restHandler{
		rawInputSchema: def.GetInputSchema(),
		inputSchema:    schema,
		endpointTmpl:   def.GetEndpoint(),
		method:         util.HttpMethodFromString(def.GetMethod(), http.MethodGet),
		headers:        def.GetHeaders(),
		body:           body,
		bodyFromInput:  bodyFromInput,
		parse:          def.GetParse(),
	}, nil
}

func (h *restHandler) GetArgsSchema() *structpb.Struct {
	return h.rawInputSchema
}

func (h *restHandler) ValidateArgs(args any) error {
	if h.inputSchema == nil {
		return errors.New("input schema cannot be nil")
	}

	mapobj, ok := args.(map[string]any)
	if !ok {
		return errors.New("args is not a map")
	}

	return schemavalidate.ValidateAgainstSchema(h.inputSchema, mapobj)
}

func (h *restHandler) ValidateUpdate(argsSchema *structpb.Struct) error {
	if argsSchema == nil {
		return errors.New("update schema cannot be nil")
	}

	if _, err := schemavalidate.CompileSchemaFromPB(argsSchema); err != nil {
		return fmt.Errorf("update validation failed due to invalid schema: %w", err)
	}
	return schemaupdate.ValidateSchemaUpdate(h.rawInputSchema, argsSchema)
}

func (h *restHandler) Call(ctx context.Context, _ *interfaces.Result, args any) (any, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return nil, errors.New("args is not a map")
	}

	expandedEndpoint, err := uritemplate.Expand(h.endpointTmpl, argsMap)
	if err != nil {
		return nil, err
	}

	transport := h.testOnlyTransport
	if transport == nil {
		transport = rego.LimitedDialer(nil)
	}
	// TODO: Add option to use custom client
	cli := &http.Client{
		// TODO: Make timeout configurable
		Timeout: 5 * time.Second,
		// Don't allow calling non-public addresses.
		Transport: transport,
	}

	b, err := h.getBody(argsMap)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, h.method, expandedEndpoint, b)
	if err != nil {
		return nil, err
	}

	for k, v := range h.headers {
		req.Header.Add(k, v)
	}

	return h.doRequest(cli, req)
}

func recordMetrics(ctx context.Context, resp *http.Response, start time.Time) {
	attrs := []attribute.KeyValue{
		attribute.String("method", resp.Request.Method),
		attribute.String("endpoint", resp.Request.URL.String()),
		attribute.String("status_code", fmt.Sprintf("%d", resp.StatusCode)),
	}

	dataSourceLatencyHistogram.Record(ctx, time.Since(start).Milliseconds(), metric.WithAttributes(attrs...))
	dataSourceRequestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (h *restHandler) doRequest(cli *http.Client, req *http.Request) (any, error) {
	start := time.Now()
	resp, err := retriableDo(cli, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	recordMetrics(req.Context(), resp, start)

	bout, err := h.parseResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	// TODO: Handle fallback here.

	return buildRestOutput(resp.StatusCode, bout), nil
}

func (h *restHandler) getBody(args map[string]any) (io.Reader, error) {
	if h.bodyFromInput {
		return h.getBodyFromInput(args)
	}

	if h.body == "" {
		return nil, nil
	}

	return strings.NewReader(h.body), nil
}

func (h *restHandler) getBodyFromInput(args map[string]any) (io.Reader, error) {
	if h.body == "" {
		return nil, errors.New("body key is empty")
	}

	body, ok := args[h.body]
	if !ok {
		return nil, fmt.Errorf("body key %q not found in args", h.body)
	}

	switch outb := body.(type) {
	case string:
		return strings.NewReader(outb), nil
	case map[string]any:
		// stringify the object
		obj, err := json.Marshal(outb)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal body object: %w", err)
		}

		return strings.NewReader(string(obj)), nil
	default:
		return nil, fmt.Errorf("body key %q is not a string or object", h.body)
	}
}

func (h *restHandler) parseResponseBody(body io.Reader) (any, error) {
	var data any

	if body == nil {
		return nil, nil
	}

	lr := io.LimitReader(body, MaxBytesLimit)

	if h.parse == "json" {
		var jsonData any
		dec := json.NewDecoder(lr)
		if err := dec.Decode(&jsonData); err != nil {
			return nil, fmt.Errorf("cannot decode json: %w", err)
		}

		data = jsonData
	} else {
		bytedata, err := io.ReadAll(lr)
		if err != nil {
			return nil, fmt.Errorf("cannot read response body: %w", err)
		}

		data = string(bytedata)
	}

	return data, nil
}

func parseRequestBodyConfig(def *minderv1.RestDataSource_Def) (bool, string) {
	defBody := def.GetBody()
	if defBody == nil {
		return false, ""
	}

	switch defBody.(type) {
	case *minderv1.RestDataSource_Def_Bodyobj:
		// stringify the object
		obj, err := json.Marshal(def.GetBodyobj())
		if err != nil {
			return false, ""
		}

		return false, string(obj)
	case *minderv1.RestDataSource_Def_BodyFromField:
		return true, def.GetBodyFromField()
	}

	return false, def.GetBodystr()
}

func buildRestOutput(statusCode int, body any) any {
	return map[string]any{
		"status_code": statusCode,
		"body":        body,
	}
}

func retriableDo(cli *http.Client, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	retryCount := 0

	err := backoff.Retry(func() error {
		var err error
		resp, err = cli.Do(req)
		if err != nil {
			zerolog.Ctx(req.Context()).Debug().
				Err(err).
				Int("retry", retryCount).
				Msg("HTTP request failed, retrying")
			retryCount++
			return err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			zerolog.Ctx(req.Context()).Debug().
				Int("retry", retryCount).
				Msg("rate limited, retrying")
			retryCount++
			return errors.New("rate limited")
		}

		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5))

	if err != nil {
		zerolog.Ctx(req.Context()).Warn().
			Err(err).
			Int("retries", retryCount).
			Msg("HTTP request failed after retries")
		return nil, err
	}

	return resp, nil
}
