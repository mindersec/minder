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

package controlplane

import (
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"

	"github.com/stacklok/minder/internal/providers/github/oauth"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const defaultProvider = oauth.Github

// HasProtoContext is an interface that can be implemented by a request
type HasProtoContext interface {
	GetContext() *pb.Context
}

// providerError wraps an error with a user visible error message
func providerError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return util.UserVisibleError(codes.NotFound, "provider not found")
	}
	return fmt.Errorf("provider error: %w", err)
}

// getNameFilterParam allows us to build a name filter for our provider queries
func getNameFilterParam(name string) sql.NullString {
	return sql.NullString{
		String: name,
		Valid:  name != "",
	}
}
