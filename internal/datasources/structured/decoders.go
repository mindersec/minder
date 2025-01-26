// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package structured

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

var (
	// ErrNoInput triggers if a decoder is called without input
	ErrNoInput = errors.New("unable to decode, no input defined")
)

// This file contains various decoders that can be used to decode structured data

// jsonDecoder decodes JSON data
type jsonDecoder struct{}

func (*jsonDecoder) Parse(r io.Reader) (any, error) {
	if r == nil {
		return nil, ErrNoInput
	}
	var res any
	dec := json.NewDecoder(r)
	if err := dec.Decode(&res); err != nil {
		return nil, fmt.Errorf("decoding json data: %w", err)
	}
	return res, nil
}

func (*jsonDecoder) Extensions() []string {
	return []string{"json"}
}

// yamlDecoder opens yaml
type yamlDecoder struct{}

func (*yamlDecoder) Parse(r io.Reader) (any, error) {
	if r == nil {
		return nil, ErrNoInput
	}
	var res any
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&res); err != nil {
		return nil, fmt.Errorf("decoding yaml data: %w", err)
	}
	return res, nil
}

func (*yamlDecoder) Extensions() []string {
	return []string{"yaml", "yml"}
}

type tomlDecoder struct{}

func (*tomlDecoder) Parse(r io.Reader) (any, error) {
	if r == nil {
		return nil, ErrNoInput
	}
	var res any
	dec := toml.NewDecoder(r)
	if err := dec.Decode(&res); err != nil {
		return nil, fmt.Errorf("decoding toml data: %w", err)
	}
	return res, nil
}

func (*tomlDecoder) Extensions() []string {
	return []string{"toml"}
}
