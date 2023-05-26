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
