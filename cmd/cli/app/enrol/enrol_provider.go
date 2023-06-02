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

package enrol

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Response struct {
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

var enrol_integrationCmd = &cobra.Command{
	Use:   "provider",
	Short: "xxx",
	Long:  `xxxx`,
	Run: func(cmd *cobra.Command, args []string) {

		providers := []string{"github"}
		if len(args) == 0 {
			fmt.Println("Please specify a provider")
			fmt.Println("Available providers:")
			for _, provider := range providers {
				fmt.Println("*  " + provider)
			}
			os.Exit(1)
		} else {
			provider := args[0]
			var validProvider bool
			for _, p := range providers {
				if p == provider {
					validProvider = true
					break
				}
			}

			if !validProvider {
				fmt.Println("Invalid provider")
				fmt.Println("Available providers:")
				for _, p := range providers {
					fmt.Println("*  " + p)
				}
				os.Exit(1)
			}
		}

		provider := args[0]
		providerURL := fmt.Sprintf("http://localhost:8080/api/v1/%s/hook", provider)

		fmt.Printf("Opening %s in your browser...\n", providerURL)
		err := browser.OpenURL(providerURL)
		if err != nil {
			fmt.Printf("Error opening browser: %v\n", err)
		}

		var wg sync.WaitGroup
		wg.Add(1)
		callBackServer(&wg)
		wg.Wait()

	},
}

func init() {
	EnrollCmd.AddCommand(enrol_integrationCmd)
	enrol_integrationCmd.Flags().String("provider", "", "The integration provider to use for login")
	if err := viper.BindPFlags(enrol_integrationCmd.PersistentFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
}
