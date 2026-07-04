// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"context"
	"strings"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/types/known/structpb"

	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

type mockDataSourceFunc struct {
	result any
}

func (*mockDataSourceFunc) ValidateArgs(_ any) error                { return nil }
func (*mockDataSourceFunc) ValidateUpdate(_ *structpb.Struct) error { return nil }
func (*mockDataSourceFunc) GetArgsSchema() *structpb.Struct         { return nil }
func (m *mockDataSourceFunc) Call(_ context.Context, _ *interfaces.Ingested, _ any) (any, error) {
	return m.result, nil
}

type mockDataSource struct {
	funcs map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef
}

func (m *mockDataSource) GetFuncs() map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef {
	return m.funcs
}

func buildMockDataSourceRegistry(dict *starlark.Dict) (*v1datasources.DataSourceRegistry, error) {
	registry := v1datasources.NewDataSourceRegistry()
	if dict == nil {
		return registry, nil
	}

	goMap, err := dictToGoMap(dict)
	if err != nil {
		return nil, err
	}

	sources := make(map[string]*mockDataSource)
	for key, val := range goMap {
		parts := strings.SplitN(key, ".", 2)
		name := parts[0]
		funcKey := ""
		if len(parts) == 2 {
			funcKey = parts[1]
		}

		if _, ok := sources[name]; !ok {
			sources[name] = &mockDataSource{
				funcs: make(map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef),
			}
		}

		sources[name].funcs[v1datasources.DataSourceFuncKey(funcKey)] = &mockDataSourceFunc{result: val}
	}

	for name, ds := range sources {
		if err := registry.RegisterDataSource(name, ds); err != nil {
			return nil, err
		}
	}

	return registry, nil
}
