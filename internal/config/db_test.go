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
	"database/sql/driver"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stacklok/minder/internal/config"
)

// TODO: add this with a fake DB connection driver
type fakeDriver struct {
	conns string
}

var _ (driver.Driver) = (*fakeDriver)(nil)
var _ (driver.DriverContext) = (*fakeDriver)(nil)
var _ (driver.Connector) = (*fakeDriver)(nil)
var _ (driver.Conn) = (*fakeDriver)(nil)

func (f fakeDriver) Open(name string) (driver.Conn, error) {
	return nil, errors.ErrUnsupported
}
func (f fakeDriver) OpenConnector(name string) (driver.Connector, error) {
	return fakeDriver{name}, errors.ErrUnsupported
}
func (f fakeDriver) Connect(ctx context.Context) (driver.Conn, error) {
	return &fakeDriver{f.conns}, nil
}
func (f fakeDriver) Driver() driver.Driver {
	return &f
}

func (f *fakeDriver) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.ErrUnsupported
}

func (f *fakeDriver) Close() error {
	return nil
}

func (f *fakeDriver) Begin() (driver.Tx, error) {
	return nil, errors.ErrUnsupported
}

func TestDatabaseConfig_GetDBConnection(t *testing.T) {
	t.Parallel()
	// sql.Register("postgres", &fakeDriver{})
	tests := []struct {
		name     string
		config   config.DatabaseConfig
		numTries int
		want     func(*testing.T, []string)
	}{{
		name: "defaults",
		want: func(t *testing.T, got []string) {
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
			urls := make([]url.URL, 0, len(got))
			for _, urlStr := range got {
				u, err := url.Parse(urlStr)
				if err != nil {
					t.Errorf("Unable to parse URL %q", urlStr)
				} else {
					urls = append(urls, *u)
				}
			}
			basePassword, _ := urls[0].User.Password()
			for i, u := range urls {
				if u.Scheme != "postgres" {
					t.Errorf("Expected postgres scheme, got %q", u.Scheme)
				}
				if u.Host != "localhost:5432" {
					t.Errorf("Unexpected host, got %q", u.Host)
				}
				if u.Path != "/minder" {
					t.Errorf("Unexpected path, got %q", u.Path)
				}
				if u.User.Username() != "postgres" {
					t.Errorf("Unexpected username, got %q", u.User.Username())
				}
				if pw, set := u.User.Password(); !set || pw == "postgres" {
					t.Errorf("Expected token, not static password")
				}
				if pw, _ := u.User.Password(); i != 0 && pw == basePassword {
					t.Errorf("Expected different token for each connection")
				}
			}
		},
	}}
	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			tt := testcase
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
				tt.config.AWSRegion = tt.config.AWSRegion
			}
			os.Setenv("AWS_ACCESS_KEY_ID", "1123")
			os.Setenv("AWS_SECRET_ACCESS_KEY", "abc")

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
