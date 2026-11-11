// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/pkg/datasources/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

type dummyFuncDef struct {
	val string
}

func (*dummyFuncDef) ValidateArgs(_ any) error {
	return nil
}

func (*dummyFuncDef) ValidateUpdate(_ *structpb.Struct) error {
	return nil
}

func (d *dummyFuncDef) Call(_ context.Context, _ *interfaces.Ingested, _ any) (any, error) {
	return d.val, nil
}

func (*dummyFuncDef) GetArgsSchema() *structpb.Struct {
	return &structpb.Struct{}
}

type dummyDataSource struct {
	funcs map[v1.DataSourceFuncKey]v1.DataSourceFuncDef
}

func (d *dummyDataSource) GetFuncs() map[v1.DataSourceFuncKey]v1.DataSourceFuncDef {
	return d.funcs
}

func TestDataSourceFuncKey_String(t *testing.T) {
	t.Parallel()

	key := v1.DataSourceFuncKey("custom_func")
	assert.Equal(t, "custom_func", key.String())
}

func TestDataSourceRegistry_RegistrationAndRetrieval(t *testing.T) {
	t.Parallel()

	t.Run("successfully registers data sources and retrieves prefixed functions", func(t *testing.T) {
		t.Parallel()

		reg := v1.NewDataSourceRegistry()
		require.NotNil(t, reg)

		ds1 := &dummyDataSource{
			funcs: map[v1.DataSourceFuncKey]v1.DataSourceFuncDef{
				"lookup":  &dummyFuncDef{val: "res_lookup"},
				"inspect": &dummyFuncDef{val: "res_inspect"},
			},
		}

		err := reg.RegisterDataSource("gh_api", ds1)
		require.NoError(t, err)

		allFuncs := reg.GetFuncs()
		assert.Len(t, allFuncs, 2)
		assert.Contains(t, allFuncs, v1.DataSourceFuncKey("gh_api.lookup"))
		assert.Contains(t, allFuncs, v1.DataSourceFuncKey("gh_api.inspect"))
	})

	t.Run("duplicate function registration across data sources returns error", func(t *testing.T) {
		t.Parallel()

		reg := v1.NewDataSourceRegistry()

		ds1 := &dummyDataSource{
			funcs: map[v1.DataSourceFuncKey]v1.DataSourceFuncDef{
				"check": &dummyFuncDef{val: "v1"},
			},
		}
		err := reg.RegisterDataSource("vuln", ds1)
		require.NoError(t, err)

		// Attempt registering same data source name and key again
		ds2 := &dummyDataSource{
			funcs: map[v1.DataSourceFuncKey]v1.DataSourceFuncDef{
				"check": &dummyFuncDef{val: "v2"},
			},
		}
		err = reg.RegisterDataSource("vuln", ds2)
		require.Error(t, err)
		assert.True(t, errors.Is(err, v1.ErrDuplicateDataSourceFuncKey))
		assert.Contains(t, err.Error(), "duplicate data source function key: vuln.check")

		// Verify existing function was retained and not overwritten
		allFuncs := reg.GetFuncs()
		require.Len(t, allFuncs, 1)
		res, callErr := allFuncs["vuln.check"].Call(context.Background(), nil, nil)
		require.NoError(t, callErr)
		assert.Equal(t, "v1", res)
	})

	t.Run("empty data source registration is a noop", func(t *testing.T) {
		t.Parallel()

		reg := v1.NewDataSourceRegistry()
		ds := &dummyDataSource{funcs: map[v1.DataSourceFuncKey]v1.DataSourceFuncDef{}}

		err := reg.RegisterDataSource("empty_ds", ds)
		require.NoError(t, err)
		assert.Empty(t, reg.GetFuncs())
	})
}

func TestDataSourceRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	reg := v1.NewDataSourceRegistry()
	var wg sync.WaitGroup

	numWriters := 20
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			dsName := "ds_" + string(rune('a'+idx))
			ds := &dummyDataSource{
				funcs: map[v1.DataSourceFuncKey]v1.DataSourceFuncDef{
					"query": &dummyFuncDef{val: dsName},
				},
			}
			_ = reg.RegisterDataSource(dsName, ds)
		}(i)
	}

	numReaders := 20
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.GetFuncs()
		}()
	}

	wg.Wait()
	assert.Len(t, reg.GetFuncs(), numWriters)
}

func TestOptions(t *testing.T) {
	t.Parallel()

	transport := &http.Transport{}
	timeout := 15 * time.Second
	maxBytes := int64(1048576)

	opts := &v1.Options{}
	optionsList := []v1.Option{
		v1.WithTestOnlyTransport(transport),
		v1.WithRESTRequestTimeout(timeout),
		v1.WithRESTMaxResponseBytes(maxBytes),
	}

	for _, opt := range optionsList {
		opt(opts)
	}

	assert.Equal(t, transport, opts.TestOnlyTransport)
	assert.Equal(t, timeout, opts.RESTRequestTimeout)
	assert.Equal(t, maxBytes, opts.RESTMaxResponseBytes)
}
