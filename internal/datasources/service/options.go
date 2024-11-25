// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import "github.com/mindersec/minder/internal/db"

// Options is a struct that contains the options for a service call
type Options struct {
	qtx db.ExtendQuerier
}

// OptionsBuilder is a function that returns a new Options struct
func OptionsBuilder() *Options {
	return &Options{}
}

// WithTransaction is a function that sets the transaction field in the Options struct
func (o *Options) WithTransaction(qtx db.ExtendQuerier) *Options {
	if o == nil {
		return nil
	}
	o.qtx = qtx
	return o
}

func (o *Options) getTransaction() db.ExtendQuerier {
	if o == nil {
		return nil
	}
	return o.qtx
}

type txGetter interface {
	getTransaction() db.ExtendQuerier
}

// ReadOptions is a struct that contains the options for a read service call
// This extends the Options struct and adds a hierarchical field.
type ReadOptions struct {
	Options
	hierarchical bool
}

// ReadBuilder is a function that returns a new ReadOptions struct
func ReadBuilder() *ReadOptions {
	return &ReadOptions{}
}

// Hierarchical allows the service to search in the project hierarchy
func (o *ReadOptions) Hierarchical() *ReadOptions {
	if o == nil {
		return nil
	}
	o.hierarchical = true
	return o
}

// WithTransaction is a function that sets the transaction field in the Options struct
func (o *ReadOptions) WithTransaction(qtx db.ExtendQuerier) *ReadOptions {
	if o == nil {
		return nil
	}
	o.qtx = qtx
	return o
}

func (o *ReadOptions) canSearchHierarchical() bool {
	if o == nil {
		return false
	}
	return o.hierarchical
}
