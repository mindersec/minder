// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// Options is a struct that contains the options for a service call
type Options struct {
	qtx      db.ExtendQuerier
	provider provinfv1.Provider
}

// OptionsBuilder is a function that returns a new Options struct
func OptionsBuilder() *Options {
	return &Options{}
}

// WithTransaction is a function that sets the transaction field in the Options struct
func (o *Options) WithTransaction(qtx db.ExtendQuerier) *Options {
	if o == nil {
		o = &Options{}
	}
	o.qtx = qtx
	return o
}

// WithProvider is a function that sets the provider field in the Options struct
func (o *Options) WithProvider(provider provinfv1.Provider) *Options {
	if o == nil {
		o = &Options{}
	}
	o.provider = provider
	return o
}

func (o *Options) getTransaction() db.ExtendQuerier {
	if o == nil {
		return nil
	}
	return o.qtx
}

func (o *Options) getProvider() provinfv1.Provider {
	if o == nil {
		return nil
	}
	return o.provider
}

type txGetter interface {
	getTransaction() db.ExtendQuerier
}

// ReadOptions is a struct that contains the options for a read service call
// This extends the Options struct and adds a hierarchical field.
type ReadOptions struct {
	Options
	hierarchical bool

	// Use the actual project hierarchy to search for the data source.
	hierarchy []uuid.UUID
}

// ReadBuilder is a function that returns a new ReadOptions struct
func ReadBuilder() *ReadOptions {
	return &ReadOptions{}
}

// Hierarchical allows the service to search in the project hierarchy
func (o *ReadOptions) Hierarchical() *ReadOptions {
	if o == nil {
		o = &ReadOptions{}
	}
	o.hierarchical = true
	return o
}

// WithTransaction is a function that sets the transaction field in the Options struct
func (o *ReadOptions) WithTransaction(qtx db.ExtendQuerier) *ReadOptions {
	if o == nil {
		o = &ReadOptions{}
	}
	o.qtx = qtx
	return o
}

// WithProvider is a function that sets the provider field in the ReadOptions struct
func (o *ReadOptions) WithProvider(provider provinfv1.Provider) *ReadOptions {
	if o == nil {
		o = &ReadOptions{}
	}
	o.provider = provider
	return o
}

// withHierarchy allows the service to search in the project hierarchy.
// This is left internal for now to disallow external use.
func (o *ReadOptions) withHierarchy(projs []uuid.UUID) *ReadOptions {
	if o == nil {
		o = &ReadOptions{}
	}
	o.hierarchy = projs
	return o
}

func (o *ReadOptions) canSearchHierarchical() bool {
	if o == nil {
		return false
	}
	return o.hierarchical
}
