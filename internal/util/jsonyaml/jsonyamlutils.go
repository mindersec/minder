// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package jsonyaml contains utility functions for converting to/from json and yaml
package jsonyaml

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type decoder interface {
	Decode(any) error
}

type encoder interface {
	Encode(any) error
}

func transcode(d decoder, e encoder) error {
	var data interface{}

	if err := d.Decode(&data); err != nil {
		return err
	}

	return e.Encode(data)
}

// TranscodeYAMLToJSON transcodes YAML to JSON
func TranscodeYAMLToJSON(r io.Reader, w io.Writer) error {
	return transcode(yaml.NewDecoder(r), json.NewEncoder(w))
}

// TranscodeJSONToYAML transcodes JSON to YAML
func TranscodeJSONToYAML(r io.Reader, w io.Writer) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return transcode(json.NewDecoder(r), enc)
}

// ConvertYamlToJson converts yaml to json
func ConvertYamlToJson(content string) (json.RawMessage, error) {
	r := strings.NewReader(content)
	w := bytes.NewBuffer(nil)

	if err := TranscodeYAMLToJSON(r, w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

// ConvertJsonToYaml converts json to yaml
func ConvertJsonToYaml(content json.RawMessage) (string, error) {
	r := bytes.NewReader(content)
	w := strings.Builder{}

	if err := TranscodeJSONToYAML(r, &w); err != nil {
		return "", err
	}

	return w.String(), nil
}
