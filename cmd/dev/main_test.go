// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mindersec/minder/cmd/dev/app"
)

func TestCobraMain(t *testing.T) {

	t.Parallel()

	// Create test files
	testDir := t.TempDir()

	// Create a simple OpenAPI v2 document with basePath
	simpleOpenAPI := `swagger: "2.0"
info:
  title: "Test API"
  version: "1.0.0"
basePath: "/api"
paths:
  /test:
    get:
      summary: "Test endpoint"
      operationId: "getTest"
      parameters: []
      responses:
        "200":
          description: "Successful response"
          content:
            application/json: {}`

	simpleOpenAPIPath := filepath.Join(testDir, "simple_openapi.json")
	err := os.WriteFile(simpleOpenAPIPath, []byte(simpleOpenAPI), 0600)
	assert.NoError(t, err, "Failed to create simple OpenAPI file")

	tests := []struct {
		name          string
		openAPIFile   string
		expectedData  string
		expectedError bool
	}{
		{
			name:        "simple API",
			openAPIFile: simpleOpenAPIPath,
			expectedData: `version: v1
		type: data-source
		context: {}
		name: Test API
		rest:
		  def:
		    get_test:
		      endpoint: /api/test
		      method: GET
		      parse: json
		      inputSchema: {}`,
			expectedError: false,
		},
		{
			name:          "missing OpenAPI file",
			openAPIFile:   filepath.Join(testDir, "missing_openapi.json"),
			expectedData:  "",
			expectedError: true,
		},
	}
	var mu sync.Mutex
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := app.CmdRoot()

			cmd.SetArgs([]string{"datasource", "generate", tt.openAPIFile})

			// Save the original os.Stdout
			originalStdout := os.Stdout

			// Create a pipe to capture the output,
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Redirect the output to the buffer in a separate goroutine
			outC := make(chan string)
			go func() {
				var buf bytes.Buffer
				_, err = io.Copy(&buf, r)
				assert.NoError(t, err, "Buffer copy should not produce an error")

				outC <- buf.String()
			}()

			// Execute command and capture any errors
			err = cmd.Execute()
			if tt.expectedError {
				assert.Error(t, err, "Expected an error but got none")
				return
			}
			// Close the writer and restore original os.Stdout
			err = w.Close()
			assert.NoError(t, err, "File close should not produce an error")
			os.Stdout = originalStdout

			// Read the captured output
			output := <-outC

			assert.NoError(t, err, "Command execution should not produce an error")

			// Handle the case for missing OpenAPI file
			if tt.name == "missing OpenAPI file" {
				mu.Lock()
				assert.FileExists(t, tt.openAPIFile, "OpenAPI file should not exist")
				mu.Unlock()
				return
			}

			assert.NoError(t, err, "Command execution should not produce an error")

			// Normalize and compare the YAML strings
			expectedYAML := strings.Join(strings.Fields(string(tt.expectedData)), "")
			generatedYAML := strings.Join(strings.Fields(string(output)), "")
			// Compare the YAML strings directly
			assert.Equal(t, expectedYAML, generatedYAML, "Generated datasource definition should match expected")

			// Add a slight delay to ensure the output is captured correctly
			time.Sleep(100 * time.Millisecond)
		})
	}
}
