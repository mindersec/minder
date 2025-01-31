// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// TestGetJsonFromProto tests the GetJsonFromProto function
func TestGetJsonFromProto(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         proto.Message
		expectedJson  string
		expectedError bool
	}{
		{
			name: "Valid proto message",
			input: &pb.Repository{
				Owner: "repoOwner",
				Name:  "repoName",
			},
			expectedJson: `{
  "owner": "repoOwner",
  "name": "repoName"
}`,
			expectedError: false,
		},
		{
			name:          "Nil proto message",
			input:         nil,
			expectedJson:  `{}`,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			jsonStr, err := util.GetJsonFromProto(tt.input)

			if (err != nil) != tt.expectedError {
				t.Errorf("GetJsonFromProto() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			// Normalize JSON strings by removing all whitespaces and new lines
			normalizedResult := strings.Join(strings.Fields(jsonStr), "")
			normalizedExpected := strings.Join(strings.Fields(tt.expectedJson), "")

			if normalizedResult != normalizedExpected {
				t.Errorf("GetJsonFromProto() = %v, expected %v", normalizedResult, normalizedExpected)
			}
		})
	}
}

// TestGetYamlFromProto tests the GetYamlFromProto function
func TestGetYamlFromProto(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         proto.Message
		expectedYaml  string
		expectedError bool
	}{
		{
			name: "Valid proto message",
			input: &pb.Repository{
				Owner: "repoOwner",
				Name:  "repoName",
			},
			expectedYaml: `name: repoName
owner: repoOwner
`,
			expectedError: false,
		},
		{
			name:  "Nil proto message",
			input: nil,
			expectedYaml: `{}
`,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			yamlResult, err := util.GetYamlFromProto(tt.input)
			if (err != nil) != tt.expectedError {
				t.Errorf("GetYamlFromProto() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			// Normalize JSON strings by removing all whitespaces and new lines
			normalizedResult := strings.Join(strings.Fields(yamlResult), "")
			normalizedExpected := strings.Join(strings.Fields(tt.expectedYaml), "")

			if normalizedResult != normalizedExpected {
				t.Errorf("GetJsonFromProto() = %v, expected %v", normalizedResult, normalizedExpected)
			}
		})
	}
}

// TestOpenFileArg tests the OpenFileArg function
func TestOpenFileArg(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	tempFilePath := filepath.Join(testDir, "testfile.txt")
	err := os.WriteFile(tempFilePath, []byte("test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testCases := []struct {
		name         string
		filePath     string
		dashOpen     io.Reader
		expectedDesc string
		expectError  bool
	}{
		{
			name:         "Dash as file path",
			filePath:     "-",
			dashOpen:     strings.NewReader("dash input"),
			expectedDesc: "dash input",
			expectError:  false,
		},
		{
			name:         "Valid file path",
			filePath:     tempFilePath,
			dashOpen:     nil,
			expectedDesc: "test content",
			expectError:  false,
		},
		{
			name:         "Invalid file path",
			filePath:     "nonexistent.txt",
			dashOpen:     nil,
			expectedDesc: "",
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			desc, closer, err := util.OpenFileArg(tc.filePath, tc.dashOpen)
			if closer != nil {
				defer closer()
			}

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				buf := new(strings.Builder)
				_, err := io.Copy(buf, desc)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedDesc, buf.String())
			}
		})
	}
}

// TestExpandFileArgs tests the ExpandFileArgs function.
func TestExpandFileArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		files    []string
		expected []util.ExpandedFile
		wantErr  bool
	}{
		{
			name:     "Single file",
			files:    []string{"testfile.txt"},
			expected: []util.ExpandedFile{{Path: "testfile.txt", Expanded: false}},
			wantErr:  false,
		},

		{
			name:     "Single directory",
			files:    []string{"testdir"},
			expected: []util.ExpandedFile{{Path: "testdir/file1.txt", Expanded: true}, {Path: "testdir/file2.txt", Expanded: true}},
			wantErr:  false,
		},
		{
			name:     "File and directory",
			files:    []string{"testfile.txt", "testdir"},
			expected: []util.ExpandedFile{{Path: "testfile.txt", Expanded: false}, {Path: "testdir/file1.txt", Expanded: true}, {Path: "testdir/file2.txt", Expanded: true}},
			wantErr:  false,
		},
		{
			name:     "File with '-'",
			files:    []string{"-"},
			expected: []util.ExpandedFile{{Path: "-", Expanded: false}},
			wantErr:  false,
		},
		{
			name:     "Non-existent file",
			files:    []string{"nonexistent.txt"},
			expected: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a temporary directory
			testDir := t.TempDir()
			if err := setupTestFiles(testDir); err != nil {
				t.Fatalf("Failed to set up test files: %v", err)
			}
			// Update file paths to include the unique directory
			for i, file := range tt.files {
				tt.files[i] = fmt.Sprintf("%s/%s", testDir, file)
			}
			for i, file := range tt.expected {
				tt.expected[i].Path = fmt.Sprintf("%s/%s", testDir, file.Path)
			}
			combinedFiles := tt.files
			got, err := util.ExpandFileArgs(combinedFiles...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandFileArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.EqualFunc(got, tt.expected, func(a, b util.ExpandedFile) bool {
				return a.Path == b.Path && a.Expanded == b.Expanded
			}) {
				t.Errorf("ExpandFileArgs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// setupTestFiles creates test files and directories for the unit tests.
func setupTestFiles(testDir string) error {
	if err := os.MkdirAll(fmt.Sprintf("%s/testdir", testDir), 0750); err != nil {
		return fmt.Errorf("failed to create directory '%s/testdir': %w", testDir, err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s/testfile.txt", testDir), []byte("test file"), 0600); err != nil {
		return fmt.Errorf("failed to create file '%s/testfile.txt': %w", testDir, err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s/testdir/file1.txt", testDir), []byte("file 1"), 0600); err != nil {
		return fmt.Errorf("failed to create file '%s/testdir/file1.txt': %w", testDir, err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s/testdir/file2.txt", testDir), []byte("file 2"), 0600); err != nil {
		return fmt.Errorf("failed to create file '%s/testdir/file2.txt': %w", testDir, err)
	}

	// Create a file named "-"
	if err := os.WriteFile(fmt.Sprintf("%s/-", testDir), []byte("dash file"), 0600); err != nil {
		return fmt.Errorf("failed to create file '%s/-': %w", testDir, err)
	}

	return nil
}

func TestGetConfigValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		key          string
		flagName     string
		defaultValue interface{}
		flagValue    interface{}
		flagSet      bool
		expected     interface{}
	}{
		{
			name:         "string flag set",
			key:          "testString",
			flagName:     "test-string",
			defaultValue: "default",
			flagValue:    "newValue",
			flagSet:      true,
			expected:     "newValue",
		},
		{
			name:         "int flag set",
			key:          "testInt",
			flagName:     "test-int",
			defaultValue: 1,
			flagValue:    42,
			flagSet:      true,
			expected:     42,
		},
		{
			name:         "flag not set",
			key:          "testFlagNotSet",
			flagName:     "test-notset",
			defaultValue: "default",
			flagValue:    "",
			flagSet:      false,
			expected:     "default",
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := viper.New()

			v.SetDefault(tc.key, tc.defaultValue)

			cmd := &cobra.Command{}
			switch tc.defaultValue.(type) {
			case string:
				cmd.Flags().String(tc.flagName, tc.defaultValue.(string), "")
			case int:
				cmd.Flags().Int(tc.flagName, tc.defaultValue.(int), "")
			}
			// bind the flag to viper
			err := v.BindPFlag(tc.key, cmd.Flags().Lookup(tc.flagName))
			if err != nil {
				t.Fatalf("Error binding flag %s: %v", tc.flagName, err)
			}
			if tc.flagSet {
				switch tc.flagValue.(type) {
				case string:
					err := cmd.Flags().Set(tc.flagName, tc.flagValue.(string))
					if err != nil {
						t.Fatalf("Error setting flag %s: %v", tc.flagName, err)
					}
				case int:
					err := cmd.Flags().Set(tc.flagName, strconv.Itoa(tc.flagValue.(int)))
					if err != nil {
						t.Fatalf("Error setting flag %s: %v", tc.flagName, err)
					}
				}
			}

			result := v.Get(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestInt32FromString(t *testing.T) {
	t.Parallel()

	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    int32
		wantErr bool
	}{
		{
			name:    "valid int32",
			args:    args{v: "42"},
			want:    42,
			wantErr: false,
		},
		{
			name:    "valid int32 negative",
			args:    args{v: "-42"},
			want:    -42,
			wantErr: false,
		},
		{
			name:    "big int32",
			args:    args{v: "2147483647"},
			want:    2147483647,
			wantErr: false,
		},
		{
			name:    "big int32 negative",
			args:    args{v: "-2147483648"},
			want:    -2147483648,
			wantErr: false,
		},
		{
			name:    "too big int32",
			args:    args{v: "12147483648"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "valid zero",
			args:    args{v: "0"},
			want:    0,
			wantErr: false,
		},
		{
			name:    "invalid int32",
			args:    args{v: "invalid"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			args:    args{v: ""},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := util.Int32FromString(tt.args.v)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Zero(t, got, "expected zero")
				return
			}

			assert.Equal(t, tt.want, got, "result didn't match")
		})
	}
}
