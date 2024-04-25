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

// Package flags containts utilities for managing feature flags.
package flags

import (
	"context"

	ofprovider "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/rs/zerolog"
	gofeature "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/retriever/fileretriever"

	"github.com/stacklok/minder/internal/auth"
	config "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/engine"
)

// Experiment is a type alias for a feature flag experiment, to ensure that all feature flags
// are registered in constants.go, not littered all over the codebase.
type Experiment string

// FromContext extracts the targeting flags from the current context.
func FromContext(ctx context.Context) openfeature.EvaluationContext {
	// Note: engine.EntityFromContext is best-effort, so these values may be zero.
	ec := engine.EntityFromContext(ctx)
	return openfeature.NewEvaluationContext(
		auth.GetUserSubjectFromContext(ctx),
		map[string]interface{}{
			"project":  ec.Project.ID.String(),
			"provider": ec.Provider.Name,
		},
	)
}

// Bool provides a simple wrapper around client.Boolean to normalize usage for Minder.
func Bool(ctx context.Context, client *openfeature.Client, feature Experiment) bool {
	ret := client.Boolean(ctx, string(feature), false, FromContext(ctx))
	// TODO: capture in telemetry records
	return ret
}

// OpenFeatureProviderFromFlags installs an OpenFeature Provider based on the flags config.
// This curently only supports the GoFeatureFlag file-based provider.
func OpenFeatureProviderFromFlags(ctx context.Context, cfg config.FlagsConfig) {
	var flagProvider openfeature.FeatureProvider
	// TODO: support relay mode by setting options.Endpoint, etc
	if cfg.GoFeature.FilePath != "" {
		zerolog.Ctx(ctx).Info().Str("path", cfg.GoFeature.FilePath).Msg("Using GoFeatureFlag file provider")
		var err error
		flagProvider, err = ofprovider.NewProvider(ofprovider.ProviderOptions{
			GOFeatureFlagConfig: &gofeature.Config{
				Retriever: &fileretriever.Retriever{
					Path: cfg.GoFeature.FilePath,
				},
			},
		})
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("Unable to create GoFeatureFlag provider")
		}
	}

	if flagProvider != nil {
		if err := openfeature.SetProvider(flagProvider); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("Unable to set flag provider, continuing without flag data")
		} else {
			zerolog.Ctx(ctx).Info().Msg("Feature flag provider installed")
		}
	} else {
		zerolog.Ctx(ctx).Warn().Msg("No feature flag provider installed")
	}
}
