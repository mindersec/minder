// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/db"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

type mockQuerier struct {
	db.ExtendQuerier
}

type mockProvider struct {
	provinfv1.Provider
}

func TestOptionsBuilder(t *testing.T) {
	t.Parallel()

	t.Run("creates empty options", func(t *testing.T) {
		t.Parallel()
		opts := OptionsBuilder()
		require.NotNil(t, opts)
		require.Nil(t, opts.getTransaction())
		require.Nil(t, opts.getProvider())
	})

	t.Run("with transaction", func(t *testing.T) {
		t.Parallel()
		opts := OptionsBuilder()
		mockTx := &mockQuerier{}
		
		ret := opts.WithTransaction(mockTx)
		require.Equal(t, opts, ret)
		require.Equal(t, mockTx, opts.getTransaction())
	})

	t.Run("with provider", func(t *testing.T) {
		t.Parallel()
		opts := OptionsBuilder()
		mockProv := &mockProvider{}
		
		ret := opts.WithProvider(mockProv)
		require.Equal(t, opts, ret)
		require.Equal(t, mockProv, opts.getProvider())
	})

	t.Run("nil receiver safety", func(t *testing.T) {
		t.Parallel()
		var opts *Options
		
		retTx := opts.WithTransaction(&mockQuerier{})
		require.NotNil(t, retTx)
		require.NotNil(t, retTx.getTransaction())

		retProv := opts.WithProvider(&mockProvider{})
		require.NotNil(t, retProv)
		require.NotNil(t, retProv.getProvider())

		require.Nil(t, opts.getTransaction())
		require.Nil(t, opts.getProvider())
	})
}

func TestReadBuilder(t *testing.T) {
	t.Parallel()

	t.Run("creates empty read options", func(t *testing.T) {
		t.Parallel()
		opts := ReadBuilder()
		require.NotNil(t, opts)
		require.False(t, opts.canSearchHierarchical())
		require.Nil(t, opts.hierarchy)
		require.Nil(t, opts.getTransaction())
		require.Nil(t, opts.getProvider())
	})

	t.Run("with transaction", func(t *testing.T) {
		t.Parallel()
		opts := ReadBuilder()
		mockTx := &mockQuerier{}
		
		ret := opts.WithTransaction(mockTx)
		require.Equal(t, opts, ret)
		require.Equal(t, mockTx, opts.getTransaction())
	})

	t.Run("with provider", func(t *testing.T) {
		t.Parallel()
		opts := ReadBuilder()
		mockProv := &mockProvider{}
		
		ret := opts.WithProvider(mockProv)
		require.Equal(t, opts, ret)
		require.Equal(t, mockProv, opts.getProvider())
	})

	t.Run("hierarchical", func(t *testing.T) {
		t.Parallel()
		opts := ReadBuilder()
		
		ret := opts.Hierarchical()
		require.Equal(t, opts, ret)
		require.True(t, opts.canSearchHierarchical())
	})

	t.Run("with hierarchy", func(t *testing.T) {
		t.Parallel()
		opts := ReadBuilder()
		hierarchy := []uuid.UUID{uuid.New(), uuid.New()}
		
		ret := opts.withHierarchy(hierarchy)
		require.Equal(t, opts, ret)
		require.Equal(t, hierarchy, opts.hierarchy)
	})

	t.Run("nil receiver safety", func(t *testing.T) {
		t.Parallel()
		var opts *ReadOptions
		
		retTx := opts.WithTransaction(&mockQuerier{})
		require.NotNil(t, retTx)
		require.NotNil(t, retTx.getTransaction())

		retProv := opts.WithProvider(&mockProvider{})
		require.NotNil(t, retProv)
		require.NotNil(t, retProv.getProvider())

		retHierarchical := opts.Hierarchical()
		require.NotNil(t, retHierarchical)
		require.True(t, retHierarchical.canSearchHierarchical())

		hierarchy := []uuid.UUID{uuid.New()}
		retHierarchy := opts.withHierarchy(hierarchy)
		require.NotNil(t, retHierarchy)
		require.Equal(t, hierarchy, retHierarchy.hierarchy)

		require.False(t, opts.canSearchHierarchical())
	})
}
