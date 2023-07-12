//
// Copyright 2023 Stacklok, Inc.
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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package util

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
	return transcode(json.NewDecoder(r), yaml.NewEncoder(w))
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
