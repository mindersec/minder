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
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Status struct {
	Status string `json:"status"`
}

// callBackServer is a simple HTTP server that listens for a callback from the
// mediators OAuth service. It will shutdown the server when the correct status
// is received and save the token to the config file.
func callBackServer(wg *sync.WaitGroup) {
	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		var status Status
		err = json.Unmarshal(body, &status)
		if err != nil {
			http.Error(w, "Error unmarshaling JSON", http.StatusBadRequest)
			return
		}

		if status.Status == "success" {
			fmt.Println("OAuth flow completed successfully")
			wg.Done() // Signal that we received the correct status and can shutdown the server.
		} else if status.Status == "failure" {
			fmt.Println("OAuth flow failed")
			wg.Done()
		} else {
			http.Error(w, "Invalid status value", http.StatusBadRequest)
		}
	})

	server := &http.Server{Addr: ":8891"}

	go func() {
		wg.Wait()
		server.Close() // Shutdown the server when the correct status is received.
	}()

	fmt.Println("Listening for OAuth Login flow to complete...")
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

// callAuthURLService calls the OAuth service to request the URL to redirect the user to.
// It accepts a provider string which is the name of the OAuth provider to use.
// For example, "google" or "github".
func callAuthURLService(address string, provider string) (string, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("error connecting to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pb.NewOAuthServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.GetAuthorizationURL(ctx, &pb.GetAuthorizationURLRequest{
		Provider: provider,
		Cli:      true,
	})

	if err != nil {
		return "", fmt.Errorf("error calling auth URL service: %v", err)
	}

	return resp.GetUrl(), nil
}

// authCmd represents the auth command
var auth_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an account in a mediator control plane",
	Long: `Create an account within a mediator control plane. Should you require 
an OAuth2 login with a provider, then pass in the --provider flag alongside
--oauth2, e.g. --oauth2 --provider=github. This will then initiate the OAuth2 flow
and allow mediator to access user account details via the provider / iDP.`,
	Run: func(cmd *cobra.Command, args []string) {

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)
		provider := util.GetConfigValue("provider", "provider", cmd, "").(string)

		url, err := callAuthURLService(fmt.Sprintf("%s:%d", grpc_host, grpc_port), provider)
		if err != nil {
			log.Fatal(err)
		}

		// Open the authorization URL in the default browser.
		fmt.Print("Opening browser to: \n", url+"\n")

		err = browser.OpenURL(url)
		if err != nil {
			log.Fatal(err)
		}

		// Start a local HTTP server to receive the callback from the mediator server.
		var wg sync.WaitGroup
		wg.Add(1)
		callBackServer(&wg)
		wg.Wait()
	},
}

func init() {
	AuthCmd.AddCommand(auth_createCmd)
	auth_createCmd.PersistentFlags().StringP("name", "n", "", "Name of the account")
	auth_createCmd.PersistentFlags().StringP("email", "e", "", "Email address of the account")
	auth_createCmd.PersistentFlags().StringP("username", "u", "", "Username of the account")
	auth_createCmd.PersistentFlags().StringP("password", "p", "", "Password of the account")
	auth_createCmd.PersistentFlags().StringP("last-name", "l", "", "Last name of the account")
	auth_createCmd.PersistentFlags().StringP("first-name", "f", "", "First name of the account")
	auth_createCmd.PersistentFlags().StringP("group-id", "g", "", "Group ID of the account to add to")
	auth_createCmd.PersistentFlags().StringP("role-id", "i", "", "Role ID of the account to add to")
	auth_createCmd.PersistentFlags().BoolP("oauth2", "o", false, "Use OAuth2 login flow (must specify --provider)")
	auth_createCmd.PersistentFlags().BoolP("active", "a", true, "Is the account active")
	auth_createCmd.PersistentFlags().StringP("provider", "r", "", "OAuth2 provider to use (e.g. google, github)")

	if err := viper.BindPFlags(auth_createCmd.PersistentFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
}
