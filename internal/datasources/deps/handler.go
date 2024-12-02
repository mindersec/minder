// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package deps implements a data source that extracts dependencies from
// a filesystem or file.
package deps

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-git/go-billy/v5/helper/iofs"
	purl "github.com/package-url/packageurl-go"
	"github.com/protobom/protobom/pkg/sbom"
	"github.com/rs/zerolog/log"

	mdeps "github.com/mindersec/minder/internal/deps"
	"github.com/mindersec/minder/internal/deps/scalibr"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
)

type depsDataSourceHandler struct {
	def       *minderv1.DepsDataSource_Def
	extractor mdeps.Extractor
}

func newHandlerFromDef(def *minderv1.DepsDataSource_Def) (*depsDataSourceHandler, error) {
	if def == nil {
		return nil, errors.New("function definition not found")
	}

	// TODO(puerco): Get extractor from type when we have other backends
	hndlr := &depsDataSourceHandler{
		extractor: scalibr.NewExtractor(),
		def:       def,
	}

	// Validate the initialization parameters
	if err := hndlr.ValidateArgs(map[string]any{
		"ecosystems": def.Ecosystems,
		"path":       def.Path,
	}); err != nil {
		return nil, fmt.Errorf("error in function definition: %w", err)
	}
	return hndlr, nil
}

func (_ *depsDataSourceHandler) ValidateArgs(args any) error {
	if args == nil {
		return nil
	}
	mapobj, ok := args.(map[string]any)
	if !ok {
		return errors.New("args is not a map")
	}

	var errs = []error{}

	// Check the known argumentss
	for k, v := range mapobj {
		switch k {
		case "ecosystems":
			errs = append(errs, validateEcosystems(v)...)
		case "path":
			if _, ok := v.(string); !ok {
				errs = append(errs, errors.New("path must be a string"))
			}
		}
	}

	return errors.Join(errs...)
}

// validateEcosystems checks that the defined ecosystems are valid
func validateEcosystems(raw any) []error {
	if raw == nil {
		return nil
	}
	ecosystems, ok := raw.([]string)
	if !ok {
		return []error{errors.New("ecosystems must be a list of strings")}
	}

	var errs = []error{}
	for _, es := range ecosystems {
		if _, ok := purl.KnownTypes[es]; !ok {
			errs = append(errs, fmt.Errorf("unkown ecosystem: %q", es))
		}
	}
	return errs
}

func (_ *depsDataSourceHandler) ValidateUpdate(_ any) error { return nil }
func (_ *depsDataSourceHandler) GetArgsSchema() any         { return nil }
func (h *depsDataSourceHandler) Call(ctx context.Context, _ any) (any, error) {
	// Extract the ingestion results from the context
	var ctxData v1datasources.Context
	var ok bool
	if ctxData, ok = ctx.Value(v1datasources.ContextKey{}).(v1datasources.Context); !ok {
		return nil, fmt.Errorf("unable to read execution context")
	}

	if ctxData.Ingest.Fs == nil {
		return nil, fmt.Errorf("filesystem not found in execution context")
	}

	nl, err := h.extractor.ScanFilesystem(ctx, iofs.New(ctxData.Ingest.Fs))
	if err != nil {
		return nil, fmt.Errorf("scanning filesystem for dependencies: %w", err)
	}

	log.Debug().Msgf("dependency extractor returned %d package nodes", len(nl.Nodes))
	nl.Nodes = append(nl.Nodes, sbom.NewNode())

	return nl, nil
}
