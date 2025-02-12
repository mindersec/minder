// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/mindersec/minder/cmd/dev/app/datasource"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

type testWriter struct {
	output string
}

func (tw *testWriter) Write(p []byte) (n int, err error) {
	tw.output += string(p)
	return len(p), nil
}

func TestCmdGenerate(t *testing.T) {
	t.Parallel()

	// Create test files
	testDir := t.TempDir()

	// Create a simple OpenAPI v2 document with basePath
	simpleOpenAPI := `{
  "swagger": "2.0",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "basePath": "/api",
  "paths": {
    "/test": {
      "get": {
        "summary": "Test endpoint",
        "operationId": "getTest",
        "parameters": [],
        "responses": {
          "200": {
            "description": "Successful response"
          }
        }
      }
    }
  }
}
`
	simpleOpenAPIPath := filepath.Join(testDir, "simple_openapi.json")
	err := os.WriteFile(simpleOpenAPIPath, []byte(simpleOpenAPI), 0600)
	assert.NoError(t, err, "Failed to create simple OpenAPI file")

	// Create the expected datasource definition for the simple API
	simpleDatasource := &minderv1.DataSource{
		Id:   uuid.New().String(),
		Name: "updated_ds",
		Context: &minderv1.ContextV2{
			ProjectId: uuid.New().String(),
		},
		Driver: &minderv1.DataSource_Rest{
			Rest: &minderv1.RestDataSource{
				Def: map[string]*minderv1.RestDataSource_Def{
					"GET_/api/test": {
						Method:   "GET",
						Endpoint: "/api/test",
						Parse:    "json",
					},
				},
			},
		},
	}

	// Marshal using protojson
	simpleDatasourceData, err := protojson.Marshal(simpleDatasource)
	assert.NoError(t, err, "Failed to marshal simple datasource data")
	simpleDatasourcePath := filepath.Join(testDir, "simple_datasource.json")
	err = os.WriteFile(simpleDatasourcePath, simpleDatasourceData, 0600)
	assert.NoError(t, err, "Failed to create simple datasource file")

	tests := []struct {
		name             string
		openAPIFile      string
		expectedDataFile string
		expectedError    bool
	}{
		{
			name:             "simple API",
			openAPIFile:      simpleOpenAPIPath,
			expectedDataFile: simpleDatasourcePath,
			expectedError:    false,
		},
		{
			name:             "missing OpenAPI file",
			openAPIFile:      filepath.Join(testDir, "missing_openapi.json"),
			expectedDataFile: "",
			expectedError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := datasource.CmdGenerate()
			tw := &testWriter{}
			cmd.SetOut(tw)
			cmd.SetErr(tw)
			cmd.SetArgs([]string{tt.openAPIFile})

			// Execute command and capture any errors
			err := cmd.Execute()
			if tt.expectedError {
				assert.Error(t, err, "Expected an error but got none")
				return
			}
			assert.NoError(t, err, "Command execution should not produce an error")

			// Handle the case for missing OpenAPI file
			if tt.name == "missing OpenAPI file" {
				assert.FileExists(t, tt.openAPIFile, "OpenAPI file should not exist")
				return
			}
			// Print captured output for debugging
			fmt.Println("Captured Output:", tw.output)
			// Load expected datasource definition
			expectedData, err := os.ReadFile(tt.expectedDataFile)
			assert.NoError(t, err, "Failed to read expected data file")

			var expectedDS minderv1.DataSource
			err = protojson.Unmarshal(expectedData, &expectedDS)
			assert.NoError(t, err, "Failed to unmarshal expected data")

			// // Load generated datasource definition
			// var generatedDS minderv1.DataSource
			// err = protojson.Unmarshal([]byte(tw.output), &generatedDS)
			// assert.NoError(t, err, "Failed to unmarshal generated data")

			// // Compare the generated and expected datasource definitions
			// if !assert.Equal(t, &expectedDS, &generatedDS, "Generated datasource definition should match expected") {
			// 	expectedStr := protojson.Format(&expectedDS)
			// 	generatedStr := protojson.Format(&generatedDS)
			// 	fmt.Printf("Mismatch between expected and generated datasource definitions:\nExpected:\n%s\nGenerated:\n%s", expectedStr, generatedStr)
			// 	t.Errorf("Buffer content: %s", tw.output)
			// }
		})
	}
}
