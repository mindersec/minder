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

package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v52/github"
)

type GitHubClient struct {
	InstallationID int64
	ApplicationID  int64
	PrivateKey     string
}

// New returns a new GitHubClient
// GitHubClient is a wrapper around the go-github client
// a := client.New(config.Github.App.Int_ID, config.Github.App.PrivateKey)
// client, _ := a.GitHubClient()
func New(applicationID int64, installationID int64, privateKey string) *GitHubClient {
	fmt.Println("New GitHubClient")
	fmt.Println("applicationID: ", applicationID)
	fmt.Println("installationID: ", installationID)
	fmt.Println("privateKey: ", privateKey)
	return &GitHubClient{
		ApplicationID:  applicationID,
		InstallationID: installationID,
		PrivateKey:     privateKey,
	}
}

// This function creates a new GitHubClient using the provided installation ID and private key.
// It uses the http.DefaultTransport to create an installation transport (itr)
// with ghinstallation.NewAppsTransport, which is then used to create a new github.Client
// and return it along with any errors encountered.
func (c *GitHubClient) GitHubClient() (*github.Client, error) {
	tr := http.DefaultTransport

	itr, err := ghinstallation.NewKeyFromFile(tr, c.ApplicationID, c.InstallationID, c.PrivateKey)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	client := github.NewClient(&http.Client{Transport: itr})
	return client, nil
}

func GetToken(appID int64, instID int64, private_key string) (string, error) {

	ctx := context.Background()

	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, instID, private_key)
	if err != nil {
		panic(err)
	}

	token, err := itr.Token(ctx)
	if err != nil {
		panic(err)
	}

	return token, nil
}
