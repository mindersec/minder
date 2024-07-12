//
// Copyright 2024 Stacklok, Inc.
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

package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ProviderConfigUnion is a union type for the different provider configurations
// this is a temporary kludge until we can autogenerate the possible attributes
type ProviderConfigUnion struct {
	*minderv1.ProviderConfig
	//nolint:lll
	GitHub *minderv1.GitHubProviderConfig `json:"github,omitempty" yaml:"github" mapstructure:"github" validate:"required"`
	//nolint:lll
	GitHubApp *minderv1.GitHubAppProviderConfig `json:"github_app,omitempty" yaml:"github_app" mapstructure:"github_app" validate:"required"`
}

// GetProviderConfig retrieves the provider configuration from the minder service
func GetProviderConfig(
	ctx context.Context,
	provCli minderv1.ProvidersServiceClient,
	project, providerName string,
) (*ProviderConfigUnion, error) {
	resp, err := provCli.GetProvider(ctx, &minderv1.GetProviderRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		Name: providerName,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}
	if resp.GetProvider() == nil {
		return nil, fmt.Errorf("could not retrieve provider, provider was empty")
	}

	provider := resp.GetProvider()
	bytes, err := provider.GetConfig().MarshalJSON()
	if err != nil {
		// TODO this is likely to be an internal error and
		// should be mapped to a more suitable user-facing
		// error.
		return nil, fmt.Errorf("error marshalling provider config: %w", err)
	}

	serde := &ProviderConfigUnion{}
	if err := json.Unmarshal(bytes, &serde); err != nil {
		// TODO this is likely to be an internal error and
		// should be mapped to a more suitable user-facing
		// error.
		return nil, fmt.Errorf("error unmarshalling provider config: %w", err)
	}

	return serde, nil
}

// SetProviderConfig sets the provider configuration in the minder service
func SetProviderConfig(
	ctx context.Context,
	provCli minderv1.ProvidersServiceClient,
	project, providerName string,
	serde *ProviderConfigUnion,
) error {
	var structConfig map[string]any

	bytes, err := json.Marshal(serde)
	if err != nil {
		// TODO this is likely to be an internal error and
		// should be mapped to a more suitable user-facing
		// error.
		return fmt.Errorf("invalid config")
	}
	if err := json.Unmarshal(bytes, &structConfig); err != nil {
		// TODO this is likely to be an internal error and
		// should be mapped to a more suitable user-facing
		// error.
		return fmt.Errorf("invalid configuration")
	}

	cfg, err := structpb.NewStruct(structConfig)
	if err != nil {
		return fmt.Errorf("invalid config patch: %w", err)
	}

	req := &minderv1.PatchProviderRequest{
		Context: &minderv1.Context{
			Project:  &project,
			Provider: &providerName,
		},
		Patch: &minderv1.Provider{
			Config: cfg,
		},
	}

	_, err = provCli.PatchProvider(ctx, req)
	if err != nil {
		return fmt.Errorf("failed calling minder: %w", err)
	}

	return nil
}
