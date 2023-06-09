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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			return &pb.LogInResponse{Status: &pb.Status{Code: int32(codes.NotFound), Message: "User and password not found"}}, nil
		}
		return nil, err
	}
	match, _ := mcrypto.VerifyPasswordHash(in.Password, user.Password)
	if err != nil {
		return &pb.LogInResponse{Status: &pb.Status{Code: int32(codes.NotFound), Message: "User and password not found"}}, nil
	}

	if !match {
		return &pb.LogInResponse{Status: &pb.Status{Code: int32(codes.NotFound), Message: "User and password not found"}}, nil
	}

	// read private key for generating token and refresh token
	privateKeyPath := viper.GetString("auth.access_token_private_key")
	if privateKeyPath == "" {
		return &pb.LogInResponse{Status: &pb.Status{Code: int32(codes.Internal), Message: "Failed to generate token"}}, nil
	}

	privateKeyPath = filepath.Clean(privateKeyPath)
	keyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return &pb.LogInResponse{Status: &pb.Status{Code: int32(codes.Internal), Message: "Failed to generate token"}}, nil
	}

	refreshPrivateKeyPath := viper.GetString("auth.refresh_token_private_key")
	if refreshPrivateKeyPath == "" {
		return &pb.LogInResponse{Status: &pb.Status{Code: int32(codes.Internal), Message: "Failed to generate token"}}, nil
	}

	refreshPrivateKeyPath = filepath.Clean(refreshPrivateKeyPath)
	refreshKeyBytes, err := ioutil.ReadFile(refreshPrivateKeyPath)
	if err != nil {
		return &pb.LogInResponse{Status: &pb.Status{Code: int32(codes.Internal), Message: "Failed to generate token"}}, nil
	}

	// read all information for user claims
	userInfo, err := s.store.GetUserClaims(ctx, user.ID)
	if err != nil {
		return &pb.LogInResponse{Status: &pb.Status{Code: int32(codes.Internal), Message: "Failed to generate token"}}, nil
	}

	claims := auth.UserClaims{
		UserId:         user.ID,
		RoleId:         userInfo.RoleID,
		GroupId:        userInfo.GroupID,
		OrganizationId: userInfo.OrganizationID,
		IsAdmin:        userInfo.IsAdmin,
		IsSuperadmin:   (userInfo.OrganizationID == 1 && userInfo.IsAdmin),
	}

	// Convert the key bytes to a string
	tokenString, refreshTokenString, tokenExpirationTime, refreshExpirationTime, err := auth.GenerateToken(
		claims,
		keyBytes,
		refreshKeyBytes,
		viper.GetInt64("auth.token_expiry"),
		viper.GetInt64("auth.refresh_expiry"),
	)

	if err != nil {
		return nil, fmt.Errorf("error generating token: %v", err)
	}

	return &pb.LogInResponse{
		Status:                &pb.Status{Code: int32(codes.OK), Message: "Success"},
		AccessToken:           tokenString,
		RefreshToken:          refreshTokenString,
		AccessTokenExpiresIn:  tokenExpirationTime,
		RefreshTokenExpiresIn: refreshExpirationTime,
	}, nil
}

// LogOut logs out a user by invalidating the access and refresh token
func (s *Server) LogOut(ctx context.Context, _ *pb.LogOutRequest) (*pb.LogOutResponse, error) {
	claims, _ := ctx.Value(TokenInfoKey).(auth.UserClaims)
	if claims.UserId > 0 {
		_, err := s.store.RevokeUserToken(ctx, claims.UserId)
		if err != nil {
			return &pb.LogOutResponse{Status: &pb.Status{Code: int32(codes.Internal), Message: "Failed to logout"}},
				status.Errorf(codes.Internal, "Failed to logout")
		}
		return &pb.LogOutResponse{Status: &pb.Status{Code: int32(codes.OK), Message: "Success"}}, nil
	}
	return &pb.LogOutResponse{Status: &pb.Status{Code: int32(codes.Internal), Message: "Failed to logout"}},
		status.Errorf(codes.Internal, "Failed to logout")
}

// RevokeTokens revokes all the access and refresh tokens
func (s *Server) RevokeTokens(ctx context.Context, _ *pb.RevokeTokensRequest) (*pb.RevokeTokensResponse, error) {
	_, err := s.store.RevokeUsersTokens(ctx)
	if err != nil {
		return &pb.RevokeTokensResponse{Status: &pb.Status{Code: int32(codes.Internal), Message: "Failed to revoke tokens"}},
			status.Errorf(codes.Internal, "Failed to revoke tokens")
	}
	return &pb.RevokeTokensResponse{Status: &pb.Status{Code: int32(codes.OK), Message: "Success"}}, nil
}

// RevokeUserToken revokes all the access and refresh tokens for a user
func (s *Server) RevokeUserToken(ctx context.Context, req *pb.RevokeUserTokenRequest) (*pb.RevokeUserTokenResponse, error) {
	_, err := s.store.RevokeUserToken(ctx, req.UserId)
	if err != nil {
		return &pb.RevokeUserTokenResponse{Status: &pb.Status{Code: int32(codes.Internal), Message: "Failed to revoke"}},
			status.Errorf(codes.Internal, "Failed to revoke")
	}
	return &pb.RevokeUserTokenResponse{Status: &pb.Status{Code: int32(codes.OK), Message: "Success"}}, nil

}
