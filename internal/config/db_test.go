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

package config_test

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stacklok/minder/internal/config"
)

// nolint: gocyclo
func TestDatabaseConfig_GetDBConnection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		config   config.DatabaseConfig
		numTries int
		want     func(*testing.T, []string)
	}{{
		name: "defaults",
		want: func(t *testing.T, got []string) {
			t.Helper()
			want := "postgres://postgres:postgres@localhost:5432/minder?sslmode=disable"
			if got[0] != want {
				t.Errorf("DatabaseConfig.GetDBConnection() = %v, want %v", got[0], want)
			}
		},
	}, {
		name: "config",
		config: config.DatabaseConfig{
			Host:     "host",
			Port:     123,
			User:     "user",
			Password: "pass",
			Name:     "database",
			SSLMode:  "enabled",
		},
		want: func(t *testing.T, got []string) {
			t.Helper()
			want := "postgres://user:pass@host:123/database?sslmode=enabled"
			if got[0] != want {
				t.Errorf("DatabaseConfig.GetDBConnection() = %v, want %v", got[0], want)
			}
		},
	}, {
		name:     "with aws, password should change over time",
		config:   config.DatabaseConfig{CloudProviderCredentials: "aws"},
		numTries: 2,
		want: func(t *testing.T, got []string) {
			t.Helper()
			urls := make([]url.URL, 0, len(got))
			for _, urlStr := range got {
				urls = append(urls, MustParseURL(urlStr))
			}
			basePassword, _ := urls[0].User.Password()
			// nolint: G101 // This one doesn't actually have a hard-coded password
			expectedNoPasswdURL := "postgres://postgres:@localhost:5432/minder?sslmode=disable"
			for i, u := range urls {
				base, pw := URLExtractPass(u)
				if i != 0 && pw == basePassword {
					t.Errorf("Expected different token for each connection")
				}
				if base.String() != expectedNoPasswdURL {
					t.Errorf("Expected base URL %q, got %q", expectedNoPasswdURL, base.String())
				}
			}
		},
	}}
	for _, testcase := range tests {
		tt := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			c := config.DefaultConfigForTest()
			if tt.config.Host != "" {
				c.Database.Host = tt.config.Host
			}
			if tt.config.Port != 0 {
				c.Database.Port = tt.config.Port
			}
			if tt.config.User != "" {
				c.Database.User = tt.config.User
			}
			if tt.config.Password != "" {
				c.Database.Password = tt.config.Password
			}
			if tt.config.Name != "" {
				c.Database.Password = tt.config.Password
			}
			if tt.config.SSLMode != "" {
				c.Database.SSLMode = tt.config.SSLMode
			}
			if tt.config.CloudProviderCredentials != "" {
				c.Database.CloudProviderCredentials = tt.config.CloudProviderCredentials
			}
			if tt.config.AWSRegion != "" {
				c.Database.AWSRegion = tt.config.AWSRegion
			}
			if os.Setenv("AWS_ACCESS_KEY_ID", "1123") != nil {
				t.Error("Unable to set env var AWS_ACCESS_KEY_ID")
			}
			if os.Setenv("AWS_SECRET_ACCESS_KEY", "abc") != nil {
				t.Error("Unable to set env var AWS_ACCESS_KEY_ID")
			}

			connStrings := make([]string, 0, tt.numTries)
			t.Logf("DB config is %v", c.Database)
			for {
				_, uri, _ := c.Database.GetDBConnection(context.Background())
				connStrings = append(connStrings, uri)
				if len(connStrings) >= tt.numTries {
					break
				}
				time.Sleep(2 * time.Second)
			}
		})
	}
}

func MustParseURL(s string) url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return *u
}

func URLExtractPass(u url.URL) (url.URL, string) {
	pw, _ := u.User.Password()
	u.User = url.UserPassword(u.User.Username(), "")
	return u, pw
}
