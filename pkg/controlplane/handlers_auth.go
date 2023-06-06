// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/auth"
	mcrypto "github.com/stacklok/mediator/pkg/crypto"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

type loginValidation struct {
	Username string `db:"username" validate:"required"`
	Password string `validate:"min=8,containsany=!@#?*"`
}

// LogIn logs in a user by verifying the username and password
func (s *Server) LogIn(ctx context.Context, in *pb.LogInRequest) (*pb.LogInResponse, error) {
	validator := validator.New()
	err := validator.Struct(loginValidation{Username: in.Username, Password: in.Password})
	if err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByUserName(ctx, in.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			return &pb.LogInResponse{Status: "User not found"}, nil
		}
		return nil, err
	}

	match, _ := mcrypto.VerifyPasswordHash(in.Password, user.Password)
	if err != nil {
		return &pb.LogInResponse{Status: "Invalid Password"}, nil
	}

	if !match {
		return &pb.LogInResponse{Status: "Invalid Password"}, nil
	}

	// read private key for generating token and refresh token
	privateKeyPath := viper.GetString("auth.access_token_private_key")
	if privateKeyPath == "" {
		return &pb.LogInResponse{Status: "Failed to read private key"}, nil
	}

	privateKeyPath = filepath.Clean(privateKeyPath)
	keyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return &pb.LogInResponse{Status: "Failed to read private key"}, nil
	}

	refreshPrivateKeyPath := viper.GetString("auth.refresh_token_private_key")
	if refreshPrivateKeyPath == "" {
		return &pb.LogInResponse{Status: "Failed to read private key"}, nil
	}

	refreshPrivateKeyPath = filepath.Clean(refreshPrivateKeyPath)
	refreshKeyBytes, err := ioutil.ReadFile(refreshPrivateKeyPath)
	if err != nil {
		return &pb.LogInResponse{Status: "Failed to read private key"}, nil
	}

	// Convert the key bytes to a string
	tokenString, refreshTokenString, tokenExpirationTime, refreshExpirationTime, err := auth.GenerateToken(
		user.ID,
		user.Username,
		keyBytes,
		refreshKeyBytes,
		viper.GetInt64("auth.token_expiry"),
		viper.GetInt64("auth.refresh_expiry"),
	)

	if err != nil {
		return nil, fmt.Errorf("error generating token: %v", err)
	}

	return &pb.LogInResponse{
		Status:                "Success",
		AccessToken:           tokenString,
		RefreshToken:          refreshTokenString,
		AccessTokenExpiresIn:  tokenExpirationTime,
		RefreshTokenExpiresIn: refreshExpirationTime,
	}, nil
}

// LogOut logs out a user by invalidating the access and refresh token
func (_ *Server) LogOut(_ context.Context, _ *pb.LogOutRequest) (*pb.LogOutResponse, error) {
	// TODO: invalidate token
	return nil, nil
}
