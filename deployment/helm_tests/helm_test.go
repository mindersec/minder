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

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

var overwrite = flag.Bool("overwrite", false, "Whether to overwrite the expected output files")

func Test_HelmValues(t *testing.T) {
	t.Parallel()

	testDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get current directory: %v", err)
	}
	yamlFiles, err := filepath.Glob("*.yaml")
	if err != nil {
		t.Fatalf("Unable to find YAML files: %v", err)
	}

	for _, yamlFile := range yamlFiles {
		filename := filepath.Join(testDir, yamlFile)
		t.Run(yamlFile, func(t *testing.T) {
			t.Parallel()

			// nolint:gosec // G204 warns of subprocess launched with variable, but we control the variable (above)
			cmd := exec.Command("helm", "template", "minder", "-f", filename, "--debug", "../helm")

			out, err := cmd.Output()
			if err != nil {
				var exit *exec.ExitError
				if errors.As(err, &exit) {
					t.Log(string(exit.Stderr))
				}
				t.Fatalf("Unable to run helm template: %v", err)
			}
			if *overwrite {
				if err := os.WriteFile(fmt.Sprintf("%s-out", filename), out, 0600); err != nil {
					t.Fatalf("Unable to write output file: %v", err)
				}
			}

			want, err := os.ReadFile(fmt.Sprintf("%s-out", filename))
			if err != nil {
				t.Fatalf("Unable to read expected file: %v", err)
			}

			if diff := cmp.Diff(string(want), string(out)); diff != "" {
				t.Errorf("Helm template output mismatch (-want +got):\n%s", diff)
			}

			// Verify that files parse as yaml.  Helm should error anyway, but this double-checks
			var value map[string]interface{}
			decoder := yaml.NewDecoder(bytes.NewReader(out))
			for ; err == nil; err = decoder.Decode(&value) {

			}
			if !errors.Is(err, io.EOF) {
				t.Errorf("Got error parsing yaml: %v", err)
			}
		})
	}
}
