//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.role/licenses/LICENSE-2.0
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

package controlplane

import (
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stacklok/mediator/pkg/util"
)

func CreateTestServer() *Server {
	// generate config file for the connection
	viper.SetConfigName("config")
	viper.AddConfigPath("../..")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		return nil
	}

	// retrieve connection string
	value := viper.AllSettings()["database"]
	databaseConfig, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	conn, err := util.GetDbConnectionFromConfig(databaseConfig)
	if err != nil {
		return nil
	}

	store := db.NewStore(conn)
	server := NewServer(store)

	return server
}
