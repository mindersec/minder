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

	mdeps "github.com/mindersec/minder/internal/deps"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
)

type depsDataSourceHandler struct {
	extractor mdeps.Extractor
}

func newHandlerFromDef(def *minderv1.DepsDataSource_Def) (*depsDataSourceHandler, error) {
	if def == nil {
		return nil, errors.New("rest data source handler definition is nil")
	}

	return &depsDataSourceHandler{}, nil
}

func (_ *depsDataSourceHandler) ValidateArgs(_ any) error { return nil }

func (_ *depsDataSourceHandler) ValidateUpdate(_ any) error { return nil }

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

	return nl, nil
}
