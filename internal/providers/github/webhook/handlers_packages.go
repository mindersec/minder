// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	entityMessage "github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/properties"
	ghprop "github.com/mindersec/minder/internal/providers/github/properties"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/db"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

// packageEvent represent any event related to a repository and one of
// its packages.
type packageEvent struct {
	Action  *string `json:"action,omitempty"`
	Repo    *repo   `json:"repository,omitempty"`
	Package *pkg    `json:"package,omitempty"`
}

type pkg struct {
	ID             *int64          `json:"id,omitempty"`
	Name           *string         `json:"name,omitempty"`
	PackageType    *string         `json:"package_type,omitempty"`
	PackageVersion *packageVersion `json:"package_version,omitempty"`
	Owner          *user           `json:"owner,omitempty"`
}

type user struct {
	ID      *int64  `json:"id,omitempty"`
	Login   *string `json:"login,omitempty"`
	HTMLURL *string `json:"html_url,omitempty"`
}

func (u *user) GetID() int64 {
	if u.ID != nil {
		return *u.ID
	}
	return 0
}

func (u *user) GetLogin() string {
	if u.Login != nil {
		return *u.Login
	}
	return ""
}

func (u *user) GetHTMLURL() string {
	if u.HTMLURL != nil {
		return *u.HTMLURL
	}
	return ""
}

type packageVersion struct {
	ID                *int64             `json:"id,omitempty"`
	Version           *string            `json:"version,omitempty"`
	ContainerMetadata *containerMetadata `json:"container_metadata,omitempty"`
}

type containerMetadata struct {
	Tag *tag `json:"tag,omitempty"`
}

type tag struct {
	Digest *string `json:"digest,omitempty"`
	Name   *string `json:"name,omitempty"`
}

func processPackageEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	l := zerolog.Ctx(ctx)

	var event *packageEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	if event.Action == nil {
		return nil, errors.New("invalid event: action is nil")
	}
	if event.Package == nil || event.Repo == nil {
		l.Info().Msg("could not determine relevant entity for event. Skipping execution.")
		return nil, errNotHandled
	}

	// We only process events "package" with action "published",
	// i.e. we do not react to action "updated".
	if *event.Action != webhookActionEventPublished {
		return nil, errNotHandled
	}

	if event.Package.Owner == nil {
		return nil, errors.New("invalid package: owner is nil")
	}

	repoProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(event.Repo.GetID()),
		properties.PropertyName:       event.Repo.GetName(),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating repository properties: %w", err)
	}
	pkgLookupProps, err := packageEventToProperties(event)
	if err != nil {
		return nil, fmt.Errorf("error converting package event to properties: %w", err)
	}

	pkgMsg := entityMessage.NewEntityRefreshAndDoMessage().
		WithEntity(pb.Entity_ENTITY_ARTIFACTS, pkgLookupProps).
		WithOriginator(pb.Entity_ENTITY_REPOSITORIES, repoProps).
		WithProviderImplementsHint(string(db.ProviderTypeGithub))

	return &processingResult{topic: constants.TopicQueueOriginatingEntityAdd, wrapper: pkgMsg}, nil
}

// This routine assumes that all necessary validation is performed on
// the upper layer and accesses package and repo without checking for
// nulls.
func packageEventToProperties(
	event *packageEvent,
) (*properties.Properties, error) {
	if event.Repo.GetFullName() == "" {
		return nil, errors.New("invalid package: full name is nil")
	}
	if event.Package.Name == nil {
		return nil, errors.New("invalid package: name is nil")
	}
	if event.Package.PackageType == nil {
		return nil, errors.New("invalid package: package type is nil")
	}

	owner := ""
	if event.Package.Owner != nil {
		owner = event.Package.Owner.GetLogin()
	}

	return properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(*event.Package.ID),
		// we need these to look up the package properties
		ghprop.ArtifactPropertyOwner: owner,
		ghprop.ArtifactPropertyName:  *event.Package.Name,
		ghprop.ArtifactPropertyType:  strings.ToLower(*event.Package.PackageType),
	})
}
