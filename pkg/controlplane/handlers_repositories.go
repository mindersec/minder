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
	"database/sql"
	"encoding/base64"
	"encoding/json"

	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/auth"
	mcrypto "github.com/stacklok/mediator/pkg/crypto"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"golang.org/x/oauth2"
)

// AddRepository adds repositories to the database and registers a webhook
func (s *Server) AddRepository(ctx context.Context,
	in *pb.AddRepositoryRequest) (*pb.AddRepositoryResponse, error) {

	claims, _ := ctx.Value((TokenInfoKey)).(auth.UserClaims)

	encToken, err := s.store.GetAccessTokenByGroupID(ctx, claims.GroupId)
	if err != nil {
		return nil, err
	}

	// base64 decode the token
	decodeToken, err := base64.StdEncoding.DecodeString(encToken.EncryptedToken)
	if err != nil {
		return nil, err
	}

	// decrypt the token
	token, err := mcrypto.DecryptBytes(viper.GetString("auth.token_key"), decodeToken)
	if err != nil {
		return nil, err
	}

	// serialise token *oauth.Token

	var decryptedToken oauth2.Token
	err = json.Unmarshal(token, &decryptedToken)
	if err != nil {
		return nil, err
	}

	// Unmarshal the in.GetRepositories() into a struct Repository
	var repositories []Repository

	for _, repository := range in.GetRepositories() {
		repositories = append(repositories, Repository{
			Owner: repository.GetOwner(),
			Repo:  repository.GetName(),
		})
	}

	registerData, err := RegisterWebHook(ctx, decryptedToken, repositories, in.Events)
	if err != nil {
		return nil, err
	}

	var results []*pb.RepositoryResult

	for _, result := range registerData {
		// Convert each result to a pb.RepositoryResult object
		pbResult := &pb.RepositoryResult{
			Owner:      result.Owner,
			Repository: result.Repository,
			HookId:     result.HookID,
			HookUrl:    result.HookURL,
			DeployUrl:  result.DeployURL,
			Success:    result.Success,
			Uuid:       result.HookUUID,
		}
		results = append(results, pbResult)

		// update the database
		_, err = s.store.CreateRepository(ctx, db.CreateRepositoryParams{
			GroupID:    claims.GroupId,
			RepoOwner:  result.Owner,
			RepoName:   result.Repository,
			WebhookID:  sql.NullInt32{Int32: int32(result.HookID), Valid: true},
			WebhookUrl: result.HookURL,
			DeployUrl:  result.DeployURL,
		})
		if err != nil {
			return nil, err
		}
	}

	response := &pb.AddRepositoryResponse{
		Results: results,
	}

	return response, nil
}
