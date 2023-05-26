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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package role

import (
	"context"
	"reflect"
	"testing"

	"github.com/stacklok/mediator/pkg/db"
)

func TestCreateRole(t *testing.T) {
	type args struct {
		ctx          context.Context
		store        db.Store
		group_id     int32
		name         string
		is_admin     *bool
		is_protected *bool
	}
	tests := []struct {
		name    string
		args    args
		want    *db.Organisation
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateRole(tt.args.ctx, tt.args.store, tt.args.group_id, tt.args.name, tt.args.is_admin, tt.args.is_protected)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRole() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateRole() = %v, want %v", got, tt.want)
			}
		})
	}
}
