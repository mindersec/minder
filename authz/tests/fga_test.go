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

// Package tests runs the fga CLI tests against each file in this directory.
// It assumes the `fga` binary is in the PATH, e.g. from `make bootstrap`
package tests

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFGA(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.tests.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no test files found")
	}
	for _, f := range files {
		file := f
		t.Run(file, func(t *testing.T) {
			t.Parallel()

			output, err := exec.Command("fga", "model", "test", "--tests", file).CombinedOutput()
			if err == nil {
				t.Logf("%s succeeded, output:\n%s", file, string(output))
				return
			}
			exit := &exec.ExitError{}
			if errors.As(err, &exit) {
				t.Errorf("%s failed with %s, output:%s\n", file, exit, string(output))
			} else {
				t.Errorf("failed to exec `fga`: %v", err)
			}
		})
	}
}