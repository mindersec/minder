// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package util provides helper functions for the minder CLI.
package util

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/itchyny/gojq"
)

// jQReadAsAny gets the values from the given accessor
// the path is the accessor path in jq format.
// the obj is the object to be evaluated using the accessor.
func jQReadAsAny(ctx context.Context, path string, obj any) (any, error) {
	out := []any{}
	accessor, err := gojq.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("data parse: cannot parse key: %w", err)
	}

	iter := accessor.RunWithContext(ctx, obj)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("error processing JQ statement: %w", err)
		}

		out = append(out, v)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no values found")
	}

	if len(out) == 1 {
		return out[0], nil
	}

	return out, nil
}

// JQEvalBoolExpression evaluates the given path on the object and returns the string value
// the path is the accessor path in jq format which must return a boolean value.
func JQEvalBoolExpression(ctx context.Context, path string, obj any) (bool, error) {
	accessor, err := gojq.Parse(path)
	if err != nil {
		return false, fmt.Errorf("data parse: cannot parse key: %w", err)
	}

	iter := accessor.RunWithContext(ctx, obj)
	v, ok := iter.Next()
	if !ok {
		// No values returned
		return false, nil
	}

	if err, ok := v.(error); ok {
		return false, fmt.Errorf("error processing JQ statement: %w", err)
	}

	// If the value is not a boolean, return an error
	if _, ok := v.(bool); !ok {
		return false, fmt.Errorf("expected boolean value, got %v", reflect.TypeOf(v))
	}

	return v.(bool), nil
}

// ErrNoValueFound is an error that is returned when the accessor doesn't find anything
var ErrNoValueFound = errors.New("evaluation error")

func newErrNoValueFound(sfmt string, args ...any) error {
	msg := fmt.Sprintf(sfmt, args...)
	return fmt.Errorf("%w: %s", ErrNoValueFound, msg)
}

// JQReadFrom gets the typed value from the given accessor. Returns an error when the accessor
// doesn't find anything or when the type assertion fails. Useful for when you know the type you're expecting
// AND the accessor must return a value (IOW, the value is required by the caller)
func JQReadFrom[T any](ctx context.Context, path string, obj any) (T, error) {
	var out T

	outAny, err := jQReadAsAny(ctx, path, obj)
	if err != nil {
		return out, err
	}

	if outAny == nil {
		return out, newErrNoValueFound("no value found for path %s", path)
	}

	// test for nil to cover the case where T is any and the accessor doesn't match - we'd attempt to type assert nil to any
	out, ok := outAny.(T)
	if !ok {
		return out, fmt.Errorf("could not type assert %v to %v", outAny, reflect.TypeOf(out))
	}

	return out, nil
}

// JQReadConstant gets the typed value from the given constant. Returns an error when the type assertion fails.
func JQReadConstant[T any](constant any) (T, error) {
	out, ok := constant.(T)
	if !ok {
		return out, fmt.Errorf("could not type assert %v to %v", constant, reflect.TypeOf(out))
	}

	return out, nil
}
