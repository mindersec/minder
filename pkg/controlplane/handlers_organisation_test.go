// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

func createServer() *Server {
	// generate config file for the connection
	configFile := util.SetupConfigFile()
	viper.SetConfigFile(configFile)
	viper.ReadInConfig()

	// retrieve connection string
	value := viper.AllSettings()["database"]
	databaseConfig, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	conn, err := util.GetDbConnectionFromConfig(databaseConfig)
	defer util.RemoveConfigFile(configFile)
	if err != nil {
		return nil
	}

	store := db.NewStore(conn)
	server := NewServer(store)
	return server
}

func TestOrganisationCreate(t *testing.T) {
	seed := time.Now().UnixNano()
	server := createServer()
	if server == nil {
		t.Fatalf("Failed to create server")
	}

	org, err := server.CreateOrganisation(context.Background(), &pb.CreateOrganisationRequest{
		Name:    util.RandomString(10, seed),
		Company: util.RandomString(10, seed),
	})

	if err != nil {
		t.Fatalf("Failed to create organisation: %v", err)
	}

	t.Logf("Created organisation: %v", org)
}
