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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/stacklok/mediator/pkg/auth"
	mcrypto "github.com/stacklok/mediator/pkg/crypto"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

type loginValidation struct {
	Username string `db:"username" validate:"required"`
	Password string `validate:"min=8,containsany=_.;?&@"`
}

func generateToken(ctx context.Context, store db.Store, userId int32) (string, string, int64, int64, auth.UserClaims, error) {
	emptyClaims := auth.UserClaims{}

	// read private key for generating token and refresh token
	privateKeyPath := viper.GetString("auth.access_token_private_key")
	if privateKeyPath == "" {
		return "", "", 0, 0, emptyClaims, fmt.Errorf("could not read private access token key")
	}

	privateKeyPath = filepath.Clean(privateKeyPath)
	keyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", "", 0, 0, emptyClaims, fmt.Errorf("failed to read access token key: %s", err)
	}

	refreshPrivateKeyPath := viper.GetString("auth.refresh_token_private_key")
	if refreshPrivateKeyPath == "" {
		return "", "", 0, 0, emptyClaims, fmt.Errorf("unable to read private refresh token key")
	}

	refreshPrivateKeyPath = filepath.Clean(refreshPrivateKeyPath)
	refreshKeyBytes, err := os.ReadFile(refreshPrivateKeyPath)
	if err != nil {
		return "", "", 0, 0, emptyClaims, fmt.Errorf("failed to read refresh token key: %s", err)
	}

	claims, err := auth.GetUserClaims(ctx, store, userId)
	if err != nil {
		return "", "", 0, 0, emptyClaims, fmt.Errorf("failed to get user claims: %s", err)
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
		return "", "", 0, 0, emptyClaims, fmt.Errorf("failed to generate token: %s", err)
	}

	return tokenString, refreshTokenString, tokenExpirationTime, refreshExpirationTime, claims, nil

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
		if errors.Is(err, sql.ErrNoRows) {
			return &pb.LogInResponse{}, status.Error(codes.NotFound, "User and password not found")
		}
		return nil, err
	}
	match, err := mcrypto.VerifyPasswordHash(in.Password, user.Password)
	if err != nil {
		return &pb.LogInResponse{}, status.Error(codes.NotFound, fmt.Sprintf("Error hashing password: %s", err))
	}

	if !match {
		return &pb.LogInResponse{}, status.Error(codes.NotFound, "Password hash does not match")
	}

	accessToken, refreshToken, accessTokenExpirationTime, refreshTokenExpirationTime,
		claims, err := generateToken(ctx, s.store, user.ID)

	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to generate token: %s", err))
	}

	// update token revoke time
	_, err = s.store.CleanTokenIat(ctx, claims.UserId)
	if err != nil {
		return nil, fmt.Errorf("error updating token revoke time: %v", err)
	}

	return &pb.LogInResponse{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresIn:  accessTokenExpirationTime,
		RefreshTokenExpiresIn: refreshTokenExpirationTime,
	}, nil
}

// LogOut logs out a user by invalidating the access and refresh token
func (s *Server) LogOut(ctx context.Context, _ *pb.LogOutRequest) (*pb.LogOutResponse, error) {
	claims, _ := ctx.Value(auth.TokenInfoKey).(auth.UserClaims)
	if claims.UserId > 0 {
		_, err := s.store.RevokeUserToken(ctx, db.RevokeUserTokenParams{ID: claims.UserId,
			MinTokenIssuedTime: sql.NullTime{Time: time.Unix(time.Now().Unix(), 0), Valid: true}})
		if err != nil {
			return &pb.LogOutResponse{}, status.Error(codes.Internal, "Failed to logout")
		}
		return &pb.LogOutResponse{}, status.Error(codes.OK, "Success")
	}
	return &pb.LogOutResponse{}, status.Error(codes.Internal, "Failed to logout")
}

// RevokeTokens revokes all the access and refresh tokens
func (s *Server) RevokeTokens(ctx context.Context, _ *pb.RevokeTokensRequest) (*pb.RevokeTokensResponse, error) {
	_, err := s.store.RevokeUsersTokens(ctx, sql.NullTime{Time: time.Unix(time.Now().Unix(), 0), Valid: true})
	if err != nil {
		return &pb.RevokeTokensResponse{}, status.Error(codes.Internal, "Failed to revoke tokens")
	}
	return &pb.RevokeTokensResponse{}, nil
}

// RevokeUserToken revokes all the access and refresh tokens for a user
func (s *Server) RevokeUserToken(ctx context.Context, req *pb.RevokeUserTokenRequest) (*pb.RevokeUserTokenResponse, error) {
	_, err := s.store.RevokeUserToken(ctx, db.RevokeUserTokenParams{ID: req.UserId,
		MinTokenIssuedTime: sql.NullTime{Time: time.Unix(time.Now().Unix(), 0), Valid: true}})
	if err != nil {
		return &pb.RevokeUserTokenResponse{}, status.Error(codes.Internal, "Failed to revoke")
	}
	return &pb.RevokeUserTokenResponse{}, nil

}

func parseRefreshToken(token string, store db.Store) (int32, error) {
	// need to read pub key from file
	publicKeyPath := viper.GetString("auth.refresh_token_public_key")
	if publicKeyPath == "" {
		return 0, fmt.Errorf("could not read refresh token public key")
	}
	pubKeyData, err := os.ReadFile(filepath.Clean(publicKeyPath))
	if err != nil {
		return 0, fmt.Errorf("failed to read refresh token public key file")
	}

	userId, err := auth.VerifyRefreshToken(token, pubKeyData, store)
	if err != nil {
		return 0, fmt.Errorf("failed to verify token: %v", err)
	}
	return userId, nil
}

// RefreshToken refreshes the access token
func (s *Server) RefreshToken(ctx context.Context, _ *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// Metadata not found
		return nil, status.Errorf(codes.Unauthenticated, "no metadata found")
	}
	refresh := ""
	if tokens := md.Get("refresh-token"); len(tokens) > 0 {
		refresh = tokens[0]
	}
	if refresh == "" {
		return nil, status.Errorf(codes.Unauthenticated, "no refresh token found")
	}

	userId, err := parseRefreshToken(refresh, s.store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// regenerate and return tokens
	accessToken, _, accessTokenExpirationTime, _, _, err := generateToken(ctx, s.store, userId)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to generate token")
	}
	return &pb.RefreshTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresIn: accessTokenExpirationTime,
	}, nil
}

// Verify verifies the access token
func (*Server) Verify(ctx context.Context, _ *pb.VerifyRequest) (*pb.VerifyResponse, error) {
	claims, _ := ctx.Value(auth.TokenInfoKey).(auth.UserClaims)
	if claims.UserId > 0 {
		return &pb.VerifyResponse{Status: "OK"}, nil
	}
	return &pb.VerifyResponse{Status: "KO"}, nil
}
