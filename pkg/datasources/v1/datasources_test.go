// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"context"
	"errors"
	"fmt"
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

	t.Run("nil data source registration returns error", func(t *testing.T) {
		t.Parallel()

		reg := v1.NewDataSourceRegistry()
		err := reg.RegisterDataSource("nil_ds", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, v1.ErrNilDataSource)
		assert.Empty(t, reg.GetFuncs())
	})

	t.Run("partial collision rolls back previously stored functions", func(t *testing.T) {
		t.Parallel()

		reg := v1.NewDataSourceRegistry()

		// Pre-register existing func "collision.fn2"
		existingDS := &dummyDataSource{
			funcs: map[v1.DataSourceFuncKey]v1.DataSourceFuncDef{
				"fn2": &dummyFuncDef{val: "original_fn2"},
			},
		}
		require.NoError(t, reg.RegisterDataSource("collision", existingDS))

		// New multi-func DS has fn1 and fn2. fn2 will collide with existingDS.
		newDS := &dummyDataSource{
			funcs: map[v1.DataSourceFuncKey]v1.DataSourceFuncDef{
				"fn1": &dummyFuncDef{val: "new_fn1"},
				"fn2": &dummyFuncDef{val: "colliding_fn2"},
			},
		}

		err := reg.RegisterDataSource("collision", newDS)
		require.Error(t, err)
		assert.ErrorIs(t, err, v1.ErrDuplicateDataSourceFuncKey)

		funcs := reg.GetFuncs()
		// fn1 must have been rolled back and not present
		_, hasFn1 := funcs["collision.fn1"]
		assert.False(t, hasFn1, "partially registered function should have been rolled back")
		// original fn2 must be retained
		require.Contains(t, funcs, v1.DataSourceFuncKey("collision.fn2"))
		res, callErr := funcs["collision.fn2"].Call(context.Background(), nil, nil)
		require.NoError(t, callErr)
		assert.Equal(t, "original_fn2", res)
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

func TestDataSourceRegistry_ConcurrentCollision(t *testing.T) {
	t.Parallel()

	reg := v1.NewDataSourceRegistry()
	var wg sync.WaitGroup
	successCount := 0
	duplicateCount := 0
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(val string) {
			defer wg.Done()
			ds := &dummyDataSource{
				funcs: map[v1.DataSourceFuncKey]v1.DataSourceFuncDef{
					"shared": &dummyFuncDef{val: val},
				},
			}
			err := reg.RegisterDataSource("atomic", ds)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successCount++
			} else if errors.Is(err, v1.ErrDuplicateDataSourceFuncKey) {
				duplicateCount++
			}
		}(fmt.Sprintf("worker_%d", i))
	}

	wg.Wait()
	assert.Equal(t, 1, successCount, "exactly one concurrent registration should succeed")
	assert.Equal(t, 49, duplicateCount, "all other concurrent registrations must fail with ErrDuplicateDataSourceFuncKey")
	assert.Len(t, reg.GetFuncs(), 1)
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
