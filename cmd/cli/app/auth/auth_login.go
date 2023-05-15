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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/stacklok/mediator/pkg/util"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Credentials struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresIn  int    `json:"access_token_expires_in"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
}

func saveCredentials(creds Credentials) (string, error) {
	// marshal the credentials to json
	credsJSON, err := json.Marshal(creds)
	if err != nil {
		return "", fmt.Errorf("error marshaling credentials: %v", err)
	}

	// Get the XDG_CONFIG_HOME environment variable
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")

	// If XDG_CONFIG_HOME is not set or empty, use $HOME/.config as the base directory
	if xdgConfigHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error getting home directory: %v", err)
		}
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

	filePath := filepath.Join(xdgConfigHome, "mediator", "credentials.json")

	err = os.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return "", fmt.Errorf("error creating directory: %v", err)
	}

	// Write the JSON data to the file
	err = os.WriteFile(filePath, credsJSON, 0600)
	if err != nil {
		return "", fmt.Errorf("error writing credentials to file: %v", err)
	}
	return filePath, nil

}

func getLoginServiceClient(ctx context.Context, address string, username string, password string, dialOptions ...grpc.DialOption) (*Credentials, error) {
	conn, err := grpc.DialContext(ctx, address, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("error connecting to server: %v", err)
	}
	defer conn.Close()

	client := pb.NewLogInServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.LogIn(ctx, &pb.LogInRequest{
		Username: username,
		Password: password,
	})

	if err != nil {
		return nil, fmt.Errorf("error logging in: %v", err)
	} else if resp.Status == "error" {
		return nil, fmt.Errorf("login service returned error status")
	}

	// marshal the credentials to json
	creds := Credentials{
		AccessToken:           resp.AccessToken,
		RefreshToken:          resp.RefreshToken,
		AccessTokenExpiresIn:  int(resp.AccessTokenExpiresIn),
		RefreshTokenExpiresIn: int(resp.RefreshTokenExpiresIn),
	}

	return &creds, nil
}

// authCmd represents the auth command
var auth_loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to mediator",
	Long:  `Login to the mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)
		username := util.GetConfigValue("username", "username", cmd, "").(string)
		password := util.GetConfigValue("password", "password", cmd, "").(string)

		address := fmt.Sprintf("%s:%d", grpc_host, grpc_port)

		ctx := context.Background()
		token, err := getLoginServiceClient(ctx, address, username, password, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println(err)
		}

		// save to file
		filePath, err := saveCredentials(*token)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("Credentials saved to %s\n", filePath)

	},
}

func init() {
	AuthCmd.AddCommand(auth_loginCmd)
	auth_loginCmd.PersistentFlags().StringP("username", "u", "", "Username to use for authentication")
	auth_loginCmd.PersistentFlags().StringP("password", "p", "", "Password to use for authentication")
	if err := viper.BindPFlags(auth_loginCmd.PersistentFlags()); err != nil {
		fmt.Println("Error binding flags:", err)
	}
}
