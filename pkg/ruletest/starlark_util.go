// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"fmt"

	"go.starlark.net/starlark"
)

// starlarkValueToGo converts a starlark.Value into its Go equivalent.
func starlarkValueToGo(val starlark.Value) (any, error) {
	switch v := val.(type) {
	case starlark.NoneType:
		return nil, nil
	case starlark.Bool:
		return bool(v), nil
	case starlark.Int:
		i, ok := v.Int64()
		if !ok {
			return nil, fmt.Errorf("starlark Int out of range")
		}
		return i, nil
	case starlark.Float:
		return float64(v), nil
	case starlark.String:
		return string(v), nil
	case *starlark.List:
		result := make([]any, v.Len())
		for i := range v.Len() {
			elem, err := starlarkValueToGo(v.Index(i))
			if err != nil {
				return nil, err
			}
			result[i] = elem
		}
		return result, nil
	case *starlark.Dict:
		result := make(map[string]any, v.Len())
		for _, item := range v.Items() {
			keyStr, ok := item[0].(starlark.String)
			if !ok {
				return nil, fmt.Errorf("dictionary key must be a string, got %s", item[0].Type())
			}
			goVal, err := starlarkValueToGo(item[1])
			if err != nil {
				return nil, fmt.Errorf("key %q: %w", keyStr, err)
			}
			result[string(keyStr)] = goVal
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported starlark type: %s", v.Type())
	}
}

// goToStarlarkValue converts a Go value into its starlark.Value equivalent.
// Unknown types are stringified as a fallback.
func goToStarlarkValue(v any) (starlark.Value, error) {
	if v == nil {
		return starlark.None, nil
	}
	switch x := v.(type) {
	case string:
		return starlark.String(x), nil
	case bool:
		return starlark.Bool(x), nil
	case int:
		return starlark.MakeInt(x), nil
	case int64:
		return starlark.MakeInt64(x), nil
	case float64:
		return starlark.Float(x), nil
	case map[string]any:
		dict := starlark.NewDict(len(x))
		for k, val := range x {
			slVal, err := goToStarlarkValue(val)
			if err != nil {
				return nil, err
			}
			if err := dict.SetKey(starlark.String(k), slVal); err != nil {
				return nil, err
			}
		}
		return dict, nil
	case []any:
		var list []starlark.Value
		for _, val := range x {
			slVal, err := goToStarlarkValue(val)
			if err != nil {
				return nil, err
			}
			list = append(list, slVal)
		}
		return starlark.NewList(list), nil
	default:
		return starlark.String(fmt.Sprintf("%v", x)), nil
	}
}

// dictToGoMap converts a starlark.Dict to map[string]any, defaulting to
// an empty map when d is nil.
func dictToGoMap(d *starlark.Dict) (map[string]any, error) {
	if d == nil {
		return map[string]any{}, nil
	}
	v, err := starlarkValueToGo(d)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map[string]any, got %T", v)
	}
	return m, nil
}
