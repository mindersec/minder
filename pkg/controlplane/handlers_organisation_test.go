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
	"reflect"
	"testing"

	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
)

func TestServer_CreateOrganisation(t *testing.T) {
	type fields struct {
		store                                  db.Store
		grpcServer                             *grpc.Server
		UnimplementedHealthServiceServer       pb.UnimplementedHealthServiceServer
		UnimplementedOAuthServiceServer        pb.UnimplementedOAuthServiceServer
		UnimplementedLogInServiceServer        pb.UnimplementedLogInServiceServer
		UnimplementedOrganisationServiceServer pb.UnimplementedOrganisationServiceServer
		OAuth2                                 *oauth2.Config
		ClientID                               string
		ClientSecret                           string
	}
	type args struct {
		ctx context.Context
		in  *pb.CreateOrganisationRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.CreateOrganisationResponse
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				store:                                  tt.fields.store,
				grpcServer:                             tt.fields.grpcServer,
				UnimplementedHealthServiceServer:       tt.fields.UnimplementedHealthServiceServer,
				UnimplementedOAuthServiceServer:        tt.fields.UnimplementedOAuthServiceServer,
				UnimplementedLogInServiceServer:        tt.fields.UnimplementedLogInServiceServer,
				UnimplementedOrganisationServiceServer: tt.fields.UnimplementedOrganisationServiceServer,
				OAuth2:                                 tt.fields.OAuth2,
				ClientID:                               tt.fields.ClientID,
				ClientSecret:                           tt.fields.ClientSecret,
			}
			got, err := s.CreateOrganisation(tt.args.ctx, tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Server.CreateOrganisation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Server.CreateOrganisation() = %v, want %v", got, tt.want)
			}
		})
	}
}
