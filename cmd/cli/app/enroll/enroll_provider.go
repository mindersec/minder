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

package enroll

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Response struct {
	Status string `json:"status"`
}

func callBackServer(port string, wg *sync.WaitGroup) {
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

		var response Response
		err = json.Unmarshal(body, &response)
		if err != nil {
			http.Error(w, "Error unmarshaling JSON", http.StatusBadRequest)
			return
		}

		if response.Status == "success" {
			fmt.Println("OAuth flow completed successfully")

			wg.Done() // Signal that we received the correct status and can shutdown the server.
		} else if response.Status == "failure" {
			fmt.Println("OAuth flow failed")
			wg.Done()
		} else {
			http.Error(w, "Invalid status value", http.StatusBadRequest)
		}
	})

	server := &http.Server{Addr: fmt.Sprintf(":%s", port)}

	go func() {
		wg.Wait()
		server.Close() // Shutdown the server when the correct status is received.
	}()

	fmt.Println("Listening for OAuth Login flow to complete on port", port)
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

var enrollProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Enroll a provider within the mediator control plane",
	Long: `The medctl enroll provider command allows a user to enroll a provider
such as GitHub into the mediator control plane. Once enrolled, users can perform
actions such as adding repositories.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := util.GetGrpcConnection(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		client := pb.NewOAuthServiceClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get random port
		port, err := util.GetRandomPort()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting random port: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.GetAuthorizationURL(ctx, &pb.GetAuthorizationURLRequest{
			Provider: "github",
			Cli:      true,
			Port:     int32(port),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting authorization URL: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Authorization URL: %s\n", resp.GetUrl())

		if err := browser.OpenURL(resp.GetUrl()); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening browser: %s\n", err)
			os.Exit(1)
		}

		var wg sync.WaitGroup
		wg.Add(1)

		go callBackServer(fmt.Sprintf("%d", port), &wg)
		wg.Wait()
	},
}

func init() {
	EnrollCmd.AddCommand(enrollProviderCmd)
	enrollProviderCmd.Flags().StringP("provider", "n", "", "Name for the organization to query")
}
