// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service

import "github.com/mindersec/minder/pkg/db"

// CallOptions is a struct that contains the options for a service call
// Since most calls will need to interact with the database, the ExtendQuerier is included
// to ensure we can pass in a transaction if needed.
type CallOptions struct {
	storeOrTransaction db.ExtendQuerier
}

// CallBuilder is a function that returns a new CallOptions struct
func CallBuilder() *CallOptions {
	return &CallOptions{}
}

// WithStoreOrTransaction is a function that sets the StoreOrTransaction field in the CallOptions struct
func (psco *CallOptions) WithStoreOrTransaction(storeOrTransaction db.ExtendQuerier) *CallOptions {
	if psco == nil {
		return nil
	}
	psco.storeOrTransaction = storeOrTransaction
	return psco
}

func (psco *CallOptions) getStoreOrTransaction() db.ExtendQuerier {
	if psco == nil {
		return nil
	}
	return psco.storeOrTransaction
}

// ReadOptions is a struct that contains the options for a read service call
// This extends the PropertiesServiceCallOptions struct and adds a TolerateStaleData field.
// This field is used to determine if the service call can return stale data or not.
// This is useful for read calls that can tolerate stale data.
type ReadOptions struct {
	CallOptions
	tolerateStaleData bool
}

// ReadBuilder is a function that returns a new ReadOptions struct
func ReadBuilder() *ReadOptions {
	return &ReadOptions{}
}

// TolerateStaleData is a function that sets the TolerateStaleData field in the ReadOptions struct
func (psco *ReadOptions) TolerateStaleData() *ReadOptions {
	if psco == nil {
		return nil
	}
	psco.tolerateStaleData = true
	return psco
}

// WithStoreOrTransaction is a function that sets the StoreOrTransaction field in the CallOptions struct
func (psco *ReadOptions) WithStoreOrTransaction(storeOrTransaction db.ExtendQuerier) *ReadOptions {
	if psco == nil {
		return nil
	}
	psco.storeOrTransaction = storeOrTransaction
	return psco
}

func (psco *ReadOptions) canTolerateStaleData() bool {
	if psco == nil {
		return false
	}
	return psco.tolerateStaleData
}

func (psco *ReadOptions) getStoreOrTransaction() db.ExtendQuerier {
	if psco == nil {
		return nil
	}
	return psco.CallOptions.getStoreOrTransaction()
}

func (psco *ReadOptions) getPropertiesServiceCallOptions() *CallOptions {
	if psco == nil {
		return nil
	}
	return &psco.CallOptions
}

type getStoreOrTransaction interface {
	getStoreOrTransaction() db.ExtendQuerier
}

func (ps *propertiesService) getStoreOrTransaction(opts getStoreOrTransaction) db.ExtendQuerier {
	if opts != nil && opts.getStoreOrTransaction() != nil {
		return opts.getStoreOrTransaction()
	}
	return ps.store
}
