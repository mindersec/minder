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
				if err := os.WriteFile(fmt.Sprintf("%s-out", filename), out, 0644); err != nil {
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
