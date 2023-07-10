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
	// "database/sql"
	"encoding/base64"

	mcrypto "github.com/stacklok/mediator/pkg/crypto"
	"github.com/stacklok/mediator/pkg/db"
	// "github.com/stacklok/mediator/pkg/db"
	"github.com/google/uuid"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateKeyPair creates a new key pair for a given group
func (s *Server) CreateKeyPair(ctx context.Context, req *pb.CreateKeyPairRequest) (*pb.CreateKeyPairResponse, error) {
	// check if user is authorized
	if !IsRequestAuthorized(ctx, req.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}
	bpass, err := base64.RawStdEncoding.DecodeString(req.Passphrase)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid passphrase")
	}

	priv, pub, err := mcrypto.GenerateKeyPair(string(bpass))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate key pair")
	}

	pHash, err := mcrypto.GeneratePasswordHash(req.Passphrase, &s.cfg.Salt)
	if err != nil {
		return nil, err
	}

	uuid_key_id := uuid.New()

	keys, err := s.store.CreateSigningKey(ctx, db.CreateSigningKeyParams{
		GroupID:       req.GroupId,
		PrivateKey:    base64.RawStdEncoding.EncodeToString(priv),
		PublicKey:     base64.RawStdEncoding.EncodeToString(pub),
		Passphrase:    pHash,
		KeyIdentifier: (uuid_key_id.String()),
	})

	return &pb.CreateKeyPairResponse{
		KeyIdentifier: keys.KeyIdentifier,
		PublicKey:     base64.RawStdEncoding.EncodeToString(pub),
	}, err
}
