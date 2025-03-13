// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package flags containts utilities for managing feature flags.
package flags

import (
	"context"

	ofprovider "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag-in-process/pkg"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/rs/zerolog"
	gofeature "github.com/thomaspoignant/go-feature-flag"
	"github.com/thomaspoignant/go-feature-flag/retriever/fileretriever"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/engine/engcontext"
	config "github.com/mindersec/minder/pkg/config/server"
)

// Experiment is a type alias for a feature flag experiment, to ensure that all feature flags
// are registered in constants.go, not littered all over the codebase.
type Experiment string

// Interface is a limited slice of openfeature.IClient, using only the methods we need.
// This prevents breakage when the openfeature.IClient interface changes.
type Interface interface {
	Boolean(ctx context.Context, key string, defaultValue bool, ec openfeature.EvaluationContext, options ...openfeature.Option) bool
}

var _ Interface = (*openfeature.Client)(nil)

// fromContext extracts the targeting flags from the current context.
func fromContext(ctx context.Context) openfeature.EvaluationContext {
	// Note: engine.EntityFromContext is best-effort, so these values may be zero.
	ec := engcontext.EntityFromContext(ctx)
	return openfeature.NewEvaluationContext(
		ec.Project.ID.String(),
		map[string]interface{}{
			"project": ec.Project.ID.String(),
			// TODO: is this useful, given how provider names are used?
			"provider": ec.Provider.Name,
			"user":     auth.IdentityFromContext(ctx).String(),
		},
	)
}

// Bool provides a simple wrapper around client.Boolean to normalize usage for Minder.
func Bool(ctx context.Context, client Interface, feature Experiment) bool {
	if client == nil {
		zerolog.Ctx(ctx).Debug().Str("flag", string(feature)).Msg("Bool called with <nil> client, returning false")
		return false
	}
	ret := client.Boolean(ctx, string(feature), false, fromContext(ctx))
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
			flagProvider = nil // Need to explicitly reset the value, see #3259
		}
	}

	if flagProvider != nil {
		if err := openfeature.SetProvider(flagProvider); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("Unable to set flag provider, continuing without flag data")
		} else {
			zerolog.Ctx(ctx).Info().Msg("Feature flag provider installed")
		}
	} else {
		if err := openfeature.SetProvider(openfeature.NoopProvider{}); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("Unable to clear flag provider")
		} else {
			zerolog.Ctx(ctx).Warn().Msg("No feature flag provider installed")
		}
	}
}
