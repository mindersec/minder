// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"

	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// RegisterDataSources implements the Eval interface.
func (e *Evaluator) RegisterDataSources(dsr *v1datasources.DataSourceRegistry) {
	e.datasources = dsr
}

// buildDataSourceOptions creates an options set from the functions available in
// a data source registry.
func buildDataSourceOptions(res *interfaces.Result, dsr *v1datasources.DataSourceRegistry) []func(*rego.Rego) {
	opts := []func(*rego.Rego){}
	if dsr == nil {
		return opts
	}

	for key, dsf := range dsr.GetFuncs() {
		opts = append(opts, buildFromDataSource(res, key, dsf))
	}

	return opts
}

// buildFromDataSource builds a rego function from a data source function.
// It takes a DataSourceFuncDef and returns a function that can be used to
// register the function with the rego engine.
func buildFromDataSource(
	res *interfaces.Result, key v1datasources.DataSourceFuncKey, dsf v1datasources.DataSourceFuncDef,
) func(*rego.Rego) {
	k := normalizeKey(key)
	return rego.Function1(
		&rego.Function{
			Name: k,
			Decl: types.NewFunction(types.Args(types.A), types.A),
		},
		func(_ rego.BuiltinContext, obj *ast.Term) (*ast.Term, error) {
			// Convert the AST value back to a Go interface{}
			jsonObj, err := ast.JSON(obj.Value)
			if err != nil {
				return nil, err
			}

			if err := dsf.ValidateArgs(jsonObj); err != nil {
				return nil, err
			}

			// Call the data source function
			ctx := context.WithValue(
				context.Background(),
				v1datasources.ContextKey{},
				v1datasources.Context{
					Ingest: res,
				},
			)
			ret, err := dsf.Call(ctx, jsonObj)
			if err != nil {
				return nil, err
			}

			val, err := ast.InterfaceToValue(ret)
			if err != nil {
				return nil, err
			}

			return ast.NewTerm(val), nil
		},
	)
}

// This converts the data source function key into a format that can be used in the rego query.
// For example, if the key is "aws.ec2.instances", it will
// be converted to "minder.data.aws.ec2.instances".
// It also normalizes the key to lowercase (which should have already been done)
// and converts any "-" to "_", finally it removes any special characters.
func normalizeKey(key v1datasources.DataSourceFuncKey) string {
	low := strings.ToLower(key.String())
	underscore := strings.ReplaceAll(low, "-", "_")
	// Remove any special characters
	norm := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '.' {
			return r
		}
		return -1
	}, underscore)
	return fmt.Sprintf("minder.datasource.%s", norm)
}