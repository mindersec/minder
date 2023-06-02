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
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"

	"github.com/spf13/viper"
	mycrypto "github.com/stacklok/mediator/pkg/crypto"
	"github.com/stacklok/mediator/pkg/ghclient"
	"github.com/stacklok/sessions"

	v52 "github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	store = sessions.NewCookieStore([]byte("super-secret-key"))
	html  = `<!DOCTYPE html>
	<html>
	<head>
		<title>Stacklok Mediator</title>
		<style>
			body {
				background-color: #000;
				color: #fff;
				font-family: Arial, sans-serif;
				text-align: center;
				margin: 0;
				padding: 0;
				height: 100vh;
				display: flex;
				flex-direction: column;
				justify-content: center;
				align-items: center;
			}
	
			h1 {
				font-size: 2.5em;
				margin-bottom: 1em;
			}
	
			p {
				font-size: 1.25em;
			}
		</style>
	</head>
	<body>
		<h1>Stacklok Mediator</h1>
		<p>Thank you for installing the Stacklok Mediator GitHub App.</p>
		<p>You may now close this page.</p>
	</body>
	</html>`
)

type ListRepositories struct {
	TotalCount   *int              `json:"total_count,omitempty"`
	Repositories []*v52.Repository `json:"repositories"`
}

// HandleGitHubAppRedirect redirects the user to the GitHub App installation page
// for the Mediator App. The User ID obtained from the JWT token is encrypted
// and stored in a secure cookie.
func (s *Server) HandleGitHubAppRedirect(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "stacklok-mediator")
	if err != nil {
		panic(err)
	}

	session.Options.MaxAge = 300
	session.Options.HttpOnly = true
	session.Options.Secure = true

	// This will be changed to the User ID obtained from the JWT token, once
	// auth is implemented. Until then we will use the default user of "1"
	userID := "1"
	encryptedUserID, err := mycrypto.EncryptRow("key", userID)
	if err != nil {
		panic(err)
	}

	encryptedUserIDBase64 := base64.StdEncoding.EncodeToString(encryptedUserID)

	session.Values["mcode"] = encryptedUserIDBase64
	err = session.Save(r, w)
	if err != nil {
		panic(err)
	}

	http.Redirect(w, r, "https://github.com/apps/stacklok-mediator/installations/new", http.StatusSeeOther)
}

// HandleGitHubAppCallback handles the callback from GitHub App installation page.
// The GitHub App installation ID is obtained from the query string and the
// encrypted user ID is retrieved from the secure cookie. The GitHub App
// installation ID and encrypted user ID are then stored in the database.
func (s *Server) HandleGitHubAppCallback(w http.ResponseWriter, r *http.Request) {
	OAuth2 := &oauth2.Config{
		ClientID:     viper.GetString("github-app.client_id"),
		ClientSecret: viper.GetString("github-app.client_secret"),
		Endpoint:     github.Endpoint,
		RedirectURL:  "http://localhost:8080/api/v1/auth/callback",
		Scopes:       []string{"user:email"},
	}
	code := r.URL.Query().Get("code")
	installationID := r.URL.Query().Get("installation_id")

	token, err := OAuth2.Exchange(context.Background(), code)
	if err != nil {
		panic(err)
	}

	fmt.Println("Access Token: ", token.AccessToken)
	fmt.Println("Installation ID", installationID)

	fmt.Fprint(w, html)

	session, err := store.Get(r, "stacklok-mediator")
	if err != nil {
		panic(err)
	}

	userID, ok := session.Values["mcode"].(string)
	if !ok {
		panic("error")
	}

	// fmt.Println("Encrypted User ID: ", encryptedUserID.(string))

	encryptedUserIDBase64, err := base64.StdEncoding.DecodeString(userID)
	if err != nil {
		fmt.Println("error decoding:", err)
	}

	// // decrypt the encryptedUserID
	userIDout, err := mycrypto.DecryptRow("key", encryptedUserIDBase64)
	if err != nil {
		fmt.Println("error:", err)
	}

	fmt.Println("Decrypted User ID: ", userIDout)

	// convert installationID to int64
	installationIDInt, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		fmt.Println("error:", err)
	}
	applicationIDInt, err := strconv.ParseInt(viper.GetString("github-app.application_id"), 10, 64)
	if err != nil {
		fmt.Println("error:", err)
	}

	a := client.New(applicationIDInt, installationIDInt, viper.GetString("github-app.private_key_path"))
	client, _ := a.GitHubClient()

	repositories, _, err := client.Apps.ListRepos(context.Background(), nil)
	if err != nil {
		fmt.Println("error:", err)
	}

	var repoList ListRepositories
	repoList.Repositories = repositories.Repositories
	for _, repo := range repoList.Repositories {
		fmt.Println("Repo Name: ", *repo.Name)
		fmt.Println("Repo ID: ", *repo.ID)
		fmt.Println("Repo Owner: ", *repo.Owner.Login)
		fmt.Println("Repo Owner ID: ", *repo.Owner.ID)
	}

	cliAppURL := "http://localhost:8891/shutdown"

	resp, err := http.Post(cliAppURL, "application/json", bytes.NewBuffer([]byte(`{"status": "success"}`)))
	if err != nil {
		fmt.Printf("failed to send status to CLI application, error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("failed to send status to CLI application, status code: %d", resp.StatusCode)
	}
}
