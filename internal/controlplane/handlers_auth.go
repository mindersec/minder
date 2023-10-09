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

	"github.com/stacklok/mediator/internal/auth"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// Verify verifies the access token
func (*Server) Verify(ctx context.Context, _ *pb.VerifyRequest) (*pb.VerifyResponse, error) {
	claims := auth.GetPermissionsFromContext(ctx)
	if claims.UserId > 0 {
		return &pb.VerifyResponse{Status: "OK"}, nil
	}
	return &pb.VerifyResponse{Status: "KO"}, nil
}
