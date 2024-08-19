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

	"github.com/puzpuzpuz/xsync/v3"
)

// Property is a struct that holds a value. It's just a wrapper around an interface{}
// with typed getters and handling of an empty receiver
type Property struct {
	value any
}

// NewProperty creates a new Property with a given value
func NewProperty(value any) *Property {
	return &Property{value: value}
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
	return propertyValueAs[bool](p.value)
}

// GetBool returns the boolean value, or false if the value is not a boolean
func (p *Property) GetBool() bool {
	val, err := p.AsBool()
	if err != nil {
		return false
	}
	return val
}

// AsString returns the string value, or an error if the value is not a string
func (p *Property) AsString() (string, error) {
	if p == nil {
		return "", fmt.Errorf("property is nil")
	}
	return propertyValueAs[string](p.value)
}

// GetString returns the string value, or an empty string if the value is not a string
func (p *Property) GetString() string {
	if p == nil {
		return ""
	}
	val, err := p.AsString()
	if err != nil {
		return ""
	}
	return val
}

// AsInt64 returns the int64 value, or an error if the value is not an int64
func (p *Property) AsInt64() (int64, error) {
	if p == nil {
		return 0, fmt.Errorf("property is nil")
	}
	return propertyValueAs[int64](p.value)
}

// GetInt64 returns the int64 value, or 0 if the value is not an int64
func (p *Property) GetInt64() int64 {
	if p == nil {
		return 0
	}
	val, err := p.AsInt64()
	if err != nil {
		return 0
	}
	return val
}

// Properties struct that holds the properties map and provides access to Property values
type Properties struct {
	props *xsync.MapOf[string, Property]
}

// NewProperties Properties from a map
func NewProperties(props map[string]any) *Properties {
	propsMap := xsync.NewMapOf[string, Property](xsync.WithPresize(len(props)))

	for key, value := range props {
		propsMap.Store(key, Property{value: value})
	}
	return &Properties{
		props: propsMap,
	}
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
