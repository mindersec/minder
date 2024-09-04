//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package properties provides a simple way to access properties of an entity
package properties

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/puzpuzpuz/xsync/v3"
	"google.golang.org/protobuf/types/known/structpb"
	"iter"
)

// Property is a struct that holds a value. It's just a wrapper around structpb.Value
// with typed getters and handling of a nil receiver
type Property struct {
	value *structpb.Value
}

const (
	typeInt64      = "int64"
	typeUint64     = "uint64"
	internalPrefix = "minder.internal."
	typeKey        = "minder.internal.type"
	valueKey       = "minder.internal.value"
)

func wrapKeyValue(key, value string) map[string]any {
	return map[string]any{
		typeKey:  key,
		valueKey: value,
	}
}

func wrapInt64(value int64) map[string]any {
	return wrapKeyValue(typeInt64, strconv.FormatInt(value, 10))
}

func wrapUint64(value uint64) map[string]any {
	return wrapKeyValue(typeUint64, strconv.FormatUint(value, 10))
}

func unwrapTypedValue(value *structpb.Value, typ string) (string, error) {
	structValue := value.GetStructValue()
	if structValue == nil {
		return "", fmt.Errorf("value is not a map")
	}

	mapValue := structValue.GetFields()
	typeVal, ok := mapValue[typeKey]
	if !ok {
		return "", fmt.Errorf("type field not found")
	}

	if typeVal.GetStringValue() != typ {
		return "", fmt.Errorf("value is not of type %s", typ)
	}

	valPayload, ok := mapValue[valueKey]
	if !ok {
		return "", fmt.Errorf("value field not found")
	}

	return valPayload.GetStringValue(), nil
}

// NewProperty creates a new Property with a given value
func NewProperty(value any) (*Property, error) {
	var err error
	var val *structpb.Value

	switch v := value.(type) {
	case int64:
		value = wrapInt64(v)
	case uint64:
		value = wrapUint64(v)
	}

	val, err = structpb.NewValue(value)
	if err != nil {
		return nil, err
	}
	return &Property{value: val}, nil
}

func propertyValueAs[T any](value any) (T, error) {
	var zero T
	val, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("value is not of type %T", zero)
	}
	return val, nil
}

// AsBool returns the boolean value, or an error if the value is not a boolean
func (p *Property) AsBool() (bool, error) {
	if p == nil {
		return false, fmt.Errorf("property is nil")
	}
	return propertyValueAs[bool](p.value.AsInterface())
}

// GetBool returns the boolean value, or false if the value is not a boolean
func (p *Property) GetBool() bool {
	if p == nil {
		return false
	}
	return p.value.GetBoolValue()
}

// AsString returns the string value, or an error if the value is not a string
func (p *Property) AsString() (string, error) {
	if p == nil {
		return "", fmt.Errorf("property is nil")
	}
	return propertyValueAs[string](p.value.AsInterface())
}

// GetString returns the string value, or an empty string if the value is not a string
func (p *Property) GetString() string {
	if p == nil {
		return ""
	}
	return p.value.GetStringValue()
}

// AsInt64 returns the int64 value, or an error if the value is not an int64
func (p *Property) AsInt64() (int64, error) {
	if p == nil {
		return 0, fmt.Errorf("property is nil")
	}

	stringVal, err := unwrapTypedValue(p.value, typeInt64)
	if err != nil {
		return 0, fmt.Errorf("failed to get int64 value: %w", err)
	}
	return strconv.ParseInt(stringVal, 10, 64)
}

// GetInt64 returns the int64 value, or 0 if the value is not an int64
func (p *Property) GetInt64() int64 {
	if p == nil {
		return 0
	}
	i64val, err := p.AsInt64()
	if err != nil {
		return 0
	}
	return i64val
}

// AsUint64 returns the uint64 value, or an error if the value is not an uint64
func (p *Property) AsUint64() (uint64, error) {
	if p == nil {
		return 0, fmt.Errorf("property is nil")
	}

	stringVal, err := unwrapTypedValue(p.value, typeUint64)
	if err != nil {
		return 0, fmt.Errorf("failed to get uint64 value: %w", err)
	}
	return strconv.ParseUint(stringVal, 10, 64)
}

// GetUint64 returns the uint64 value, or 0 if the value is not an uint64
func (p *Property) GetUint64() uint64 {
	if p == nil {
		return 0
	}
	u64val, err := p.AsUint64()
	if err != nil {
		return 0
	}
	return u64val
}

// RawValue returns the raw value as an any
func (p *Property) RawValue() any {
	if p == nil {
		return nil
	}
	return p.value.AsInterface()
}

// Properties struct that holds the properties map and provides access to Property values
type Properties struct {
	props *xsync.MapOf[string, Property]
}

// NewProperties Properties from a map
func NewProperties(props map[string]any) (*Properties, error) {
	propsMap := xsync.NewMapOf[string, Property](xsync.WithPresize(len(props)))

	for key, value := range props {
		if strings.HasPrefix(key, internalPrefix) {
			return nil, fmt.Errorf("property key %s is reserved", key)
		}

		propVal, err := NewProperty(value)
		if err != nil {
			return nil, fmt.Errorf("failed to create property for key %s: %w", key, err)
		}
		propsMap.Store(key, *propVal)
	}
	return &Properties{
		props: propsMap,
	}, nil
}

// GetProperty returns the Property for a given key or an empty one as a fallback
func (p *Properties) GetProperty(key string) *Property {
	if p == nil {
		return nil
	}

	prop, ok := p.props.Load(key)
	if !ok {
		return nil
	}
	return &prop
}

// Iterate implements the seq2 iterator so that the caller can call for key, prop := range Iterate()
func (p *Properties) Iterate() iter.Seq2[string, *Property] {
	return func(yield func(string, *Property) bool) {
		p.props.Range(func(key string, v Property) bool {
			return yield(key, &v)
		})
	}
}

// PropertyFilter is a function that filters properties
type PropertyFilter func(key string, prop *Property) bool

// FilteredCopy returns a new Properties with only the properties that pass the filter
func (p *Properties) FilteredCopy(filter PropertyFilter) *Properties {
	if p == nil {
		return nil
	}

	propsMap := xsync.NewMapOf[string, Property]()
	p.props.Range(func(key string, prop Property) bool {
		if filter(key, &prop) {
			propsMap.Store(key, prop)
		}
		return true
	})

	return &Properties{
		props: propsMap,
	}
}

// Merge merges two Properties into a new one
func (p *Properties) Merge(other *Properties) *Properties {
	if p == nil {
		return other
	}

	if other == nil {
		return p
	}

	propsMap := xsync.NewMapOf[string, Property](xsync.WithPresize(p.props.Size() + other.props.Size()))
	p.props.Range(func(key string, prop Property) bool {
		propsMap.Store(key, prop)
		return true
	})

	other.props.Range(func(key string, prop Property) bool {
		propsMap.Store(key, prop)
		return true
	})

	return &Properties{
		props: propsMap,
	}
}

// ToProtoStruct converts the Properties to a protobuf Struct
func (p *Properties) ToProtoStruct() *structpb.Struct {
	if p == nil {
		return nil
	}

	fields := make(map[string]*structpb.Value)

	p.props.Range(func(key string, prop Property) bool {
		fields[key] = prop.value
		return true
	})

	protoStruct := &structpb.Struct{
		Fields: fields,
	}

	return protoStruct
}
