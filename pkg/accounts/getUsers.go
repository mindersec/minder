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

package accounts

import (
	// "bufio"
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
)

type UserInfo interface {
	GetID() int64
	GetSub() string
	GetLogin() string
	GetName() string
	GetEmail() string
	GetAvatarURL() string
}

type GithubUser struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Company   string `json:"company"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type GoogleUser struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Profile       string `json:"profile"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// Common mapping functions for Github and Google user info
func (g *GithubUser) GetLogin() string     { return g.Login }
func (g *GithubUser) GetID() int64         { return g.ID }
func (g *GithubUser) GetSub() string       { return "" }
func (g *GithubUser) GetName() string      { return g.Name }
func (g *GithubUser) GetEmail() string     { return g.Email }
func (g *GithubUser) GetAvatarURL() string { return g.AvatarURL }

func (g *GoogleUser) GetLogin() string     { return "" }
func (g *GoogleUser) GetID() int64         { return 0 }
func (g *GoogleUser) GetSub() string       { return g.Sub }
func (g *GoogleUser) GetName() string      { return g.Name }
func (g *GoogleUser) GetEmail() string     { return g.Email }
func (g *GoogleUser) GetAvatarURL() string { return g.Picture }

func GetUserInfo(oauthConfig *oauth2.Config, token *oauth2.Token, provider string) (UserInfo, error) {
	var user UserInfo
	switch provider {
	case "google":
		googleUser, e := getGoogleUserInfo(oauthConfig, token)
		if e != nil {
			return nil, e
		}
		user = googleUser
	case "github":
		githubUser, e := getGitHubUserInfo(oauthConfig, token)
		if e != nil {
			return nil, e
		}
		user = githubUser
	default:
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}
	return user, nil
}

func getGitHubUserInfo(OAuth2 *oauth2.Config, token *oauth2.Token) (*GithubUser, error) {

	client := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user GithubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func getGoogleUserInfo(OAuth2 *oauth2.Config, token *oauth2.Token) (*GoogleUser, error) {

	client := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user GoogleUser

	// print key, value pairs
	// scanner := bufio.NewScanner(resp.Body)
	// for scanner.Scan() {
	// 	fmt.Println(scanner.Text())

	// }

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}
